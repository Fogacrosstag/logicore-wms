package dev.logicore.warehouse.domain;

import dev.logicore.common.BaseEntity;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "racks")
public class Rack extends BaseEntity {
  public UUID zoneId;
  public String code;
}
