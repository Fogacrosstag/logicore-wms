package dev.logicore.reservation.api;

import dev.logicore.common.*;
import dev.logicore.reservation.application.ReservationService;
import dev.logicore.reservation.domain.Reservation;
import jakarta.validation.Valid;
import java.util.*;
import org.springframework.web.bind.annotation.*;

@RestController
public class ReservationController {
  private final ReservationService service;
  private final Repository repo;
  private final Commands commands;

  public ReservationController(ReservationService s, Repository r, Commands c) {
    service = s;
    repo = r;
    commands = c;
  }

  @PostMapping("/reservations")
  @ResponseStatus(org.springframework.http.HttpStatus.ACCEPTED)
  public Object create(
      @RequestHeader("Idempotency-Key") String k, @Valid @RequestBody ReservationService.Input q) {
    return Api.data(commands.run(k, List.of("create", q), () -> service.create(q)));
  }

  @GetMapping("/reservations/{id}")
  public Object get(@PathVariable UUID id) {
    return Api.data(repo.get(Reservation.class, id));
  }

  @DeleteMapping("/reservations/{id}")
  @ResponseStatus(org.springframework.http.HttpStatus.ACCEPTED)
  public Object cancel(@PathVariable UUID id, @RequestHeader("Idempotency-Key") String k) {
    return Api.data(commands.run(k, List.of("cancel", id), () -> service.cancel(id)));
  }
}
