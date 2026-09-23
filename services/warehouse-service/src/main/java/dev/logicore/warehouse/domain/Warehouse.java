package dev.logicore.warehouse.domain;

import dev.logicore.common.BaseEntity;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "warehouses")
public class Warehouse extends BaseEntity {
  public String code;
  public String name;
  public String address;
  public String status;
}
