package dev.logicore.common;

import java.util.Map;
import java.util.concurrent.TimeUnit;
import org.apache.kafka.clients.admin.Admin;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.actuate.health.*;
import org.springframework.stereotype.Component;

@Component
public class KafkaHealth implements HealthIndicator {
  private final String brokers;

  public KafkaHealth(@Value("${spring.kafka.bootstrap-servers}") String b) {
    brokers = b;
  }

  public Health health() {
    try (var admin =
        Admin.create(
            Map.of(
                "bootstrap.servers",
                brokers,
                "request.timeout.ms",
                2000,
                "default.api.timeout.ms",
                2000))) {
      admin.describeCluster().clusterId().get(2, TimeUnit.SECONDS);
      return Health.up().build();
    } catch (Exception e) {
      return Health.down().withDetail("dependency", "kafka").build();
    }
  }
}
