package dev.logicore.shipment.domain;

import dev.logicore.common.BaseEntity;
import dev.logicore.common.Line;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "shipments")
public class Shipment extends BaseEntity {
  public UUID warehouseId;
  public UUID reservationId;
  public String status;

  @org.hibernate.annotations.JdbcTypeCode(org.hibernate.type.SqlTypes.JSON)
  public List<Line> items;

  public String reason;
}
