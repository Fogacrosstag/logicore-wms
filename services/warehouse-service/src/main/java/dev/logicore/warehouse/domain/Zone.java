package dev.logicore.warehouse.domain;

import dev.logicore.common.BaseEntity;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "zones")
public class Zone extends BaseEntity {
  public UUID warehouseId;
  public String code;
  public String name;
  public String type;
}
