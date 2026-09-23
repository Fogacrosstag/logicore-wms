package dev.logicore.reservation.application;

import dev.logicore.common.*;
import dev.logicore.reservation.domain.Reservation;
import jakarta.validation.Valid;
import jakarta.validation.constraints.*;
import java.time.Instant;
import java.util.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@Transactional
public class ReservationService implements EventHandler {
  public record Input(
      @NotNull UUID warehouseId,
      @NotEmpty List<@Valid Line> items,
      @Min(1) @Max(1800) Integer ttlSeconds) {}

  private final Repository repo;
  private final Events events;

  public ReservationService(Repository r, Events e) {
    repo = r;
    events = e;
  }

  public Reservation create(Input q) {
    Set<UUID> products = new HashSet<>();
    for (var l : q.items())
      if (!products.add(l.productId())) throw new IllegalArgumentException("duplicate product");
    Reservation r = new Reservation();
    r.warehouseId = q.warehouseId();
    r.items = q.items();
    r.status = "PENDING";
    r.expiresAt = Instant.now().plusSeconds(q.ttlSeconds() == null ? 1800 : q.ttlSeconds());
    repo.add(r);
    events.emit(
        "ReservationRequested",
        Map.of(
            "id",
            r.id,
            "reservationId",
            r.id,
            "warehouseId",
            r.warehouseId,
            "items",
            r.items,
            "expiresAt",
            r.expiresAt));
    return r;
  }

  public Reservation cancel(UUID id) {
    var r = repo.lock(Reservation.class, id);
    Fault.require(
        !Set.of("COMPLETED", "EXPIRED", "REJECTED").contains(r.status),
        "INVALID_STATE",
        "Reservation cannot be cancelled");
    if (!r.status.equals("CANCELLED")) {
      events.emit(
          "ReservationCancellationRequested",
          Map.of("reservationId", id, "warehouseId", r.warehouseId));
      r.status = "CANCELLING";
    }
    return r;
  }

  public void handle(Event e) {
    String next =
        switch (e.eventType()) {
          case "StockReserved" -> "CONFIRMED";
          case "ReservationRejected" -> "REJECTED";
          case "StockReservationCancelled" -> "CANCELLED";
          case "ReservationExpired" -> "EXPIRED";
          case "StockShipped" -> "COMPLETED";
          default -> null;
        };
    if (next == null) return;
    var r =
        repo.lock(Reservation.class, UUID.fromString(e.payload().get("reservationId").toString()));
    if (Set.of("COMPLETED", "CANCELLED", "EXPIRED", "REJECTED").contains(r.status)) return;
    if (r.status.equals("CANCELLING") && next.equals("CONFIRMED")) return;
    r.status = next;
    r.reason = Objects.toString(e.payload().get("reason"), null);
  }
}
