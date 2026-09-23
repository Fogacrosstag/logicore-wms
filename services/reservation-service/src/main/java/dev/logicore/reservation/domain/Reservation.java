package dev.logicore.reservation.domain;

import dev.logicore.common.BaseEntity;
import dev.logicore.common.Line;
import jakarta.persistence.*;
import java.time.Instant;
import java.util.*;

@Entity
@Table(name = "reservations")
public class Reservation extends BaseEntity {
  public UUID warehouseId;
  public String status;
  public Instant expiresAt;

  @org.hibernate.annotations.JdbcTypeCode(org.hibernate.type.SqlTypes.JSON)
  public List<Line> items;

  public String reason;
}
