package dev.logicore.shipment.application;

import dev.logicore.common.*;
import dev.logicore.shipment.domain.Shipment;
import jakarta.validation.constraints.*;
import java.util.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.client.RestClient;

@Service
@Transactional
public class ShipmentService implements EventHandler {
  public record Input(@NotNull UUID reservationId) {}

  private final Repository repo;
  private final Events events;
  private final RestClient client;

  public ShipmentService(
      Repository r,
      Events e,
      RestClient.Builder b,
      @Value("${RESERVATION_URL:http://reservation-service:8080}") String url) {
    repo = r;
    events = e;
    client = b.baseUrl(url).build();
  }

  @SuppressWarnings("unchecked")
  public Shipment create(Input q) {
    Map<String, Object> response;
    try {
      response = client.get().uri("/reservations/" + q.reservationId()).retrieve().body(Map.class);
    } catch (Exception e) {
      throw new Fault("DEPENDENCY_UNAVAILABLE", "Reservation lookup failed", 503);
    }
    var r = (Map<String, Object>) response.get("data");
    Fault.require(
        "CONFIRMED".equals(r.get("status")),
        "RESERVATION_REQUIRED",
        "Confirmed reservation required");
    Shipment s = new Shipment();
    s.reservationId = q.reservationId();
    s.warehouseId = UUID.fromString(r.get("warehouseId").toString());
    s.status = "RESERVED";
    s.items =
        ((List<Map<String, Object>>) r.get("items"))
            .stream()
                .map(
                    l ->
                        new Line(
                            UUID.fromString(l.get("productId").toString()),
                            ((Number) l.get("quantity")).longValue()))
                .toList();
    repo.add(s);
    events.emit(
        "ShipmentCreated",
        Map.of(
            "shipmentId",
            s.id,
            "warehouseId",
            s.warehouseId,
            "reservationId",
            s.reservationId,
            "items",
            s.items));
    return s;
  }

  public Shipment transition(UUID id, String action) {
    var s = repo.lock(Shipment.class, id);
    switch (action) {
      case "start-picking":
        StateMachine.require(s.status, "RESERVED");
        s.status = "PICKING";
        break;
      case "pack":
        StateMachine.require(s.status, "PICKING");
        s.status = "PACKED";
        break;
      case "ready":
        StateMachine.require(s.status, "PACKED");
        s.status = "READY_FOR_SHIPMENT";
        break;
      case "ship":
        Fault.require(
            Set.of("PACKED", "READY_FOR_SHIPMENT").contains(s.status),
            "INVALID_STATE",
            "Shipment must be packed");
        s.status = "SHIPPING";
        events.emit(
            "ShipmentRequested",
            Map.of(
                "shipmentId", id, "warehouseId", s.warehouseId, "reservationId", s.reservationId));
        break;
      case "cancel":
        Fault.require(
            !Set.of("SHIPPING", "SHIPPED", "CANCELLED").contains(s.status),
            "INVALID_STATE",
            "Shipment cannot be cancelled");
        s.status = "CANCELLING";
        events.emit(
            "ReservationCancellationRequested",
            Map.of(
                "shipmentId", id, "warehouseId", s.warehouseId, "reservationId", s.reservationId));
        break;
      default:
        throw new Fault("NOT_FOUND", "Unknown shipment action", 404);
    }
    events.emit(
        "ShipmentStatusChanged",
        Map.of("shipmentId", id, "warehouseId", s.warehouseId, "status", s.status));
    return s;
  }

  public void handle(Event e) {
    var p = e.payload();
    if (e.eventType().equals("StockShipped")) {
      var s = repo.lock(Shipment.class, UUID.fromString(p.get("shipmentId").toString()));
      if (s.status.equals("SHIPPED")) return;
      s.status = "SHIPPED";
      events.emit(
          "ShipmentCompleted",
          Map.of(
              "shipmentId",
              s.id,
              "warehouseId",
              s.warehouseId,
              "reservationId",
              s.reservationId,
              "items",
              s.items));
    } else if (Set.of("ShipmentRejected", "StockReservationCancelled", "ReservationExpired")
        .contains(e.eventType())) {
      var rows =
          repo.by(
              Shipment.class, "reservationId", UUID.fromString(p.get("reservationId").toString()));
      for (var row : rows) {
        var s = repo.lock(Shipment.class, row.id);
        if (s.status.equals("SHIPPED")) continue;
        s.status = e.eventType().equals("ShipmentRejected") ? "FAILED" : "CANCELLED";
        s.reason = Objects.toString(p.get("reason"), e.eventType());
      }
    }
  }
}
