package dev.logicore.warehouse.api;

import dev.logicore.common.*;
import dev.logicore.warehouse.application.WarehouseService;
import dev.logicore.warehouse.domain.*;
import jakarta.validation.Valid;
import java.util.*;
import org.springframework.web.bind.annotation.*;

@RestController
public class WarehouseController {
  private final WarehouseService service;
  private final Repository repo;
  private final Commands commands;

  public WarehouseController(WarehouseService s, Repository r, Commands c) {
    service = s;
    repo = r;
    commands = c;
  }

  @PostMapping("/warehouses")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object create(
      @RequestHeader("Idempotency-Key") String k,
      @Valid @RequestBody WarehouseService.WarehouseInput q) {
    return Api.data(commands.run(k, List.of("warehouse", q), () -> service.create(q)));
  }

  @GetMapping("/warehouses")
  public Object warehouses() {
    return Api.data(repo.list(Warehouse.class));
  }

  @GetMapping("/warehouses/{id}")
  public Object warehouse(@PathVariable UUID id) {
    return Api.data(repo.get(Warehouse.class, id));
  }

  @PostMapping("/warehouses/{id}/zones")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object zone(
      @PathVariable UUID id,
      @RequestHeader("Idempotency-Key") String k,
      @Valid @RequestBody WarehouseService.ZoneInput q) {
    return Api.data(commands.run(k, List.of("zone", id, q), () -> service.zone(id, q)));
  }

  @GetMapping("/warehouses/{id}/zones")
  public Object zones(@PathVariable UUID id) {
    return Api.data(repo.by(Zone.class, "warehouseId", id));
  }

  @PostMapping("/zones/{id}/racks")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object rack(
      @PathVariable UUID id,
      @RequestHeader("Idempotency-Key") String k,
      @Valid @RequestBody WarehouseService.RackInput q) {
    return Api.data(commands.run(k, List.of("rack", id, q), () -> service.rack(id, q)));
  }

  @GetMapping("/zones/{id}/racks")
  public Object racks(@PathVariable UUID id) {
    return Api.data(repo.by(Rack.class, "zoneId", id));
  }

  @PostMapping("/racks/{id}/locations")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object location(
      @PathVariable UUID id,
      @RequestHeader("Idempotency-Key") String k,
      @Valid @RequestBody WarehouseService.LocationInput q) {
    return Api.data(commands.run(k, List.of("location", id, q), () -> service.location(id, q)));
  }

  @GetMapping("/racks/{id}/locations")
  public Object locations(@PathVariable UUID id) {
    return Api.data(repo.by(Location.class, "rackId", id));
  }

  @GetMapping("/warehouses/{id}/locations")
  public Object warehouseLocations(@PathVariable UUID id) {
    return Api.data(repo.by(Location.class, "warehouseId", id));
  }

  @GetMapping("/locations/{id}")
  public Object location(@PathVariable UUID id) {
    return Api.data(repo.get(Location.class, id));
  }

  @PatchMapping("/locations/{id}/status")
  public Object status(
      @PathVariable UUID id,
      @RequestHeader("Idempotency-Key") String k,
      @RequestBody Map<String, String> q) {
    return Api.data(
        commands.run(k, List.of("status", id, q), () -> service.status(id, q.get("status"))));
  }
}
