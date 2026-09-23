package dev.logicore.warehouse.application;

import dev.logicore.common.*;
import dev.logicore.warehouse.domain.*;
import jakarta.validation.constraints.*;
import java.util.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@Transactional
public class WarehouseService implements EventHandler {
  public record WarehouseInput(
      @NotBlank String code, @NotBlank String name, @NotBlank String address) {}

  public record ZoneInput(@NotBlank String code, @NotBlank String name, @NotBlank String type) {}

  public record RackInput(@NotBlank String code) {}

  public record LocationInput(
      @NotBlank String code,
      @Positive double maxWeight,
      @Positive double maxVolume,
      @PositiveOrZero double distance,
      @NotBlank String storageType) {}

  private final Repository repo;
  private final Events events;

  public WarehouseService(Repository r, Events e) {
    repo = r;
    events = e;
  }

  public Warehouse create(WarehouseInput q) {
    Warehouse x = new Warehouse();
    x.code = q.code();
    x.name = q.name();
    x.address = q.address();
    x.status = "ACTIVE";
    repo.add(x);
    events.emit("WarehouseCreated", Map.of("id", x.id, "warehouseId", x.id));
    return x;
  }

  public Zone zone(UUID id, ZoneInput q) {
    repo.get(Warehouse.class, id);
    if (!Set.of("RECEIVING", "STORAGE", "PICKING", "PACKING", "SHIPPING", "QUARANTINE")
        .contains(q.type())) throw new IllegalArgumentException("zone type");
    Zone z = new Zone();
    z.warehouseId = id;
    z.code = q.code();
    z.name = q.name();
    z.type = q.type();
    repo.add(z);
    events.emit("WarehouseZoneCreated", Map.of("id", z.id, "warehouseId", id));
    return z;
  }

  public Rack rack(UUID id, RackInput q) {
    var z = repo.get(Zone.class, id);
    Rack r = new Rack();
    r.zoneId = id;
    r.code = q.code();
    repo.add(r);
    events.emit("RackCreated", Map.of("id", r.id, "warehouseId", z.warehouseId));
    return r;
  }

  public Location location(UUID id, LocationInput q) {
    var rack = repo.get(Rack.class, id);
    var zone = repo.get(Zone.class, rack.zoneId);
    if (!Set.of("NORMAL", "FRAGILE", "COLD", "HAZARDOUS", "OVERSIZED").contains(q.storageType()))
      throw new IllegalArgumentException("storage type");
    Location l = new Location();
    l.rackId = id;
    l.warehouseId = zone.warehouseId;
    l.zoneType = zone.type;
    l.code = q.code();
    l.maxWeight = q.maxWeight();
    l.maxVolume = q.maxVolume();
    l.distance = q.distance();
    l.storageType = q.storageType();
    l.status = "AVAILABLE";
    repo.add(l);
    events.emit("WarehouseLocationCreated", Map.of("id", l.id, "warehouseId", l.warehouseId));
    return l;
  }

  public Location status(UUID id, String status) {
    var l = repo.lock(Location.class, id);
    if (!Set.of("AVAILABLE", "BLOCKED", "MAINTENANCE").contains(status))
      throw new IllegalArgumentException("status");
    l.status = status.equals("AVAILABLE") ? occupancy(l) : status;
    events.emit(
        "WarehouseLocationStatusChanged",
        Map.of("id", id, "warehouseId", l.warehouseId, "status", l.status));
    return l;
  }

  private String occupancy(Location l) {
    return l.currentWeight >= l.maxWeight || l.currentVolume >= l.maxVolume
        ? "FULL"
        : l.currentWeight > 0 || l.currentVolume > 0 ? "PARTIALLY_OCCUPIED" : "AVAILABLE";
  }

  public void handle(Event e) {
    if (!e.eventType().equals("LocationCapacityChanged")) return;
    var p = e.payload();
    Location l = repo.lock(Location.class, UUID.fromString(p.get("locationId").toString()));
    long revision = ((Number) p.get("revision")).longValue();
    if (revision <= l.occupancyRevision) return;
    l.currentWeight = ((Number) p.get("currentWeight")).doubleValue();
    l.currentVolume = ((Number) p.get("currentVolume")).doubleValue();
    l.occupancyRevision = revision;
    if (!Set.of("BLOCKED", "MAINTENANCE").contains(l.status)) l.status = occupancy(l);
  }
}
