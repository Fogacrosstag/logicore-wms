package dev.logicore.shipment.api;

import dev.logicore.common.*;
import dev.logicore.shipment.application.ShipmentService;
import dev.logicore.shipment.domain.Shipment;
import jakarta.validation.Valid;
import java.util.*;
import org.springframework.web.bind.annotation.*;

@RestController
public class ShipmentController {
  private final ShipmentService service;
  private final Repository repo;
  private final Commands commands;

  public ShipmentController(ShipmentService s, Repository r, Commands c) {
    service = s;
    repo = r;
    commands = c;
  }

  @PostMapping("/shipments")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object create(
      @RequestHeader("Idempotency-Key") String k, @Valid @RequestBody ShipmentService.Input q) {
    return Api.data(commands.run(k, List.of("create", q), () -> service.create(q)));
  }

  @GetMapping("/shipments/{id}")
  public Object get(@PathVariable UUID id) {
    return Api.data(repo.get(Shipment.class, id));
  }

  @PostMapping("/shipments/{id}/{action}")
  public Object transition(
      @PathVariable UUID id,
      @PathVariable String action,
      @RequestHeader("Idempotency-Key") String k) {
    return Api.data(commands.run(k, List.of(action, id), () -> service.transition(id, action)));
  }
}
