package dev.logicore.common;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.micrometer.core.instrument.MeterRegistry;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.TimeUnit;
import org.slf4j.MDC;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class Events {
  private final JdbcTemplate db;
  private final ObjectMapper json;
  private final KafkaTemplate<String, String> kafka;
  private final org.springframework.beans.factory.ObjectProvider<EventHandler> handler;
  private final String name;
  private final MeterRegistry metrics;
  private final io.micrometer.tracing.Tracer tracer;

  public Events(
      JdbcTemplate d,
      ObjectMapper j,
      KafkaTemplate<String, String> k,
      org.springframework.beans.factory.ObjectProvider<EventHandler> h,
      @Value("${spring.application.name}") String n,
      MeterRegistry m,
      io.micrometer.tracing.Tracer tracer) {
    db = d;
    json = j;
    kafka = k;
    handler = h;
    name = n;
    metrics = m;
    this.tracer = tracer;
  }

  public void emit(String type, Map<String, Object> payload) {
    try {
      var e =
          new Event(
              UUID.randomUUID(),
              type,
              1,
              Instant.now(),
              name,
              Objects.toString(MDC.get("correlationId"), UUID.randomUUID().toString()),
              traceparent(),
              payload);
      db.update(
          "insert into outbox_events(id,aggregate_id,event_type,payload) values(?,?,?,?::jsonb)",
          e.eventId(),
          Objects.toString(
              payload.getOrDefault("warehouseId", payload.getOrDefault("id", e.eventId()))),
          type,
          json.writeValueAsString(e));
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  @Scheduled(fixedDelay = 300)
  @Transactional
  public void publish() {
    var rows =
        db.queryForList(
            "select id,aggregate_id,payload::text from outbox_events where published_at is null"
                + " order by sequence limit 50 for update skip locked");
    for (var row : rows) {
      try {
        kafka
            .send("wms.events", row.get("aggregate_id").toString(), row.get("payload").toString())
            .get(10, TimeUnit.SECONDS);
        db.update("update outbox_events set published_at=now() where id=?", row.get("id"));
      } catch (Exception e) {
        throw new IllegalStateException("Outbox delivery failed; transaction will retry", e);
      }
    }
  }

  @KafkaListener(topics = "wms.events")
  @Transactional
  public void consume(String raw) {
    try {
      Event e = json.readValue(raw, Event.class);
      if (e.eventVersion() != 1) throw new IllegalArgumentException("Unsupported event version");
      MDC.put("correlationId", e.correlationId());
      MDC.put("traceparent", e.traceparent());
      if (db.update(
              "insert into processed_events(event_id,consumer) values(?,?) on conflict do nothing",
              e.eventId(),
              name)
          == 0) return;
      var builder = tracer.spanBuilder().name("consume " + e.eventType());
      if (e.traceparent() != null
          && e.traceparent().matches("00-[a-f0-9]{32}-[a-f0-9]{16}-[a-f0-9]{2}")) {
        var parts = e.traceparent().split("-");
        builder.setParent(
            tracer.traceContextBuilder().traceId(parts[1]).spanId(parts[2]).sampled(true).build());
      }
      var span = builder.start();
      try (var scope = tracer.withSpan(span)) {
        handler.getObject().handle(e);
        metrics.counter("kafka_messages_processed_total").increment();
      } finally {
        span.end();
      }
    } catch (RuntimeException e) {
      throw e;
    } catch (Exception e) {
      throw new IllegalStateException(e);
    } finally {
      MDC.remove("correlationId");
      MDC.remove("traceparent");
    }
  }

  private String traceparent() {
    var span = tracer.currentSpan();
    return span == null
        ? Objects.toString(MDC.get("traceparent"), "")
        : "00-" + span.context().traceId() + "-" + span.context().spanId() + "-01";
  }
}
