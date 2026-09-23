package dev.logicore.product.domain;

import dev.logicore.common.BaseEntity;
import jakarta.persistence.*;
import java.util.*;

@Entity
@Table(name = "products")
public class Product extends BaseEntity {
  public String sku;
  public String name;
  public String description;
  public String category;
  public String barcode;
  public double weight;
  public double length;
  public double width;
  public double height;
  public double volume;
  public String storageType;
  public boolean active;
}
