package dev.logicore.reservation;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import dev.logicore.common.*;
import dev.logicore.reservation.application.ReservationService;
import dev.logicore.reservation.domain.Reservation;
import java.time.Instant;
import java.util.*;
import org.junit.jupiter.api.Test;

class ReservationServiceTest {
  @Test
  void lateConfirmationCannotUndoCancellation() {
    var repo = mock(Repository.class);
    var service = new ReservationService(repo, mock(Events.class));
    var r = new Reservation();
    r.status = "CANCELLING";
    when(repo.lock(Reservation.class, r.id)).thenReturn(r);
    service.handle(
        new Event(
            UUID.randomUUID(),
            "StockReserved",
            1,
            Instant.now(),
            "inventory-service",
            "test",
            "",
            Map.of("reservationId", r.id.toString())));
    assertEquals("CANCELLING", r.status);
    service.handle(
        new Event(
            UUID.randomUUID(),
            "StockReservationCancelled",
            1,
            Instant.now(),
            "inventory-service",
            "test",
            "",
            Map.of("reservationId", r.id.toString())));
    assertEquals("CANCELLED", r.status);
  }

  @Test
  void duplicateProductsRejectedBeforePublishing() {
    var events = mock(Events.class);
    var service = new ReservationService(mock(Repository.class), events);
    var product = UUID.randomUUID();
    assertThrows(
        IllegalArgumentException.class,
        () ->
            service.create(
                new ReservationService.Input(
                    UUID.randomUUID(), List.of(new Line(product, 1), new Line(product, 2)), 30)));
    verifyNoInteractions(events);
  }
}
