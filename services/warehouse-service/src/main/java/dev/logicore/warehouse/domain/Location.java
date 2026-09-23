package dev.logicore.warehouse.domain;

import dev.logicore.common.BaseEntity;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "locations")
public class Location extends BaseEntity {
  public UUID rackId;
  public UUID warehouseId;
  public String code;
  public double maxWeight;
  public double maxVolume;
  public double currentWeight;
  public double currentVolume;
  public double distance;
  public String storageType;
  public String zoneType;
  public String status;
  public long occupancyRevision;
}
