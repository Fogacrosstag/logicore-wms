package dev.logicore.product.application;

import dev.logicore.common.*;
import dev.logicore.product.domain.Product;
import jakarta.validation.constraints.*;
import java.util.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@Transactional
public class ProductService implements EventHandler {
  public record Input(
      @NotBlank String sku,
      @NotBlank String name,
      String description,
      String category,
      String barcode,
      @Positive double weight,
      @Positive double length,
      @Positive double width,
      @Positive double height,
      @NotBlank String storageType) {}

  private final Repository repo;
  private final Events events;
  private final ProductMapper mapper;

  public ProductService(Repository r, Events e, ProductMapper m) {
    repo = r;
    events = e;
    mapper = m;
  }

  public Product create(Input q) {
    validate(q);
    Product p = new Product();
    p.sku = q.sku();
    p.weight = q.weight();
    p.length = q.length();
    p.width = q.width();
    p.height = q.height();
    p.volume = q.length() * q.width() * q.height();
    p.storageType = q.storageType();
    p.active = true;
    mapper.patch(q, p);
    repo.add(p);
    events.emit("ProductCreated", Map.of("id", p.id, "productId", p.id, "sku", p.sku));
    return p;
  }

  public Product update(UUID id, Input q) {
    validate(q);
    Product p = repo.lock(Product.class, id);
    Fault.require(
        p.sku.equals(q.sku())
            && p.weight == q.weight()
            && p.length == q.length()
            && p.width == q.width()
            && p.height == q.height()
            && p.storageType.equals(q.storageType()),
        "IMMUTABLE_STOCK_PROPERTIES",
        "Create a new SKU to change dimensions, weight or storage type");
    mapper.patch(q, p);
    events.emit("ProductUpdated", Map.of("id", id, "productId", id));
    return p;
  }

  public Product delete(UUID id) {
    Product p = repo.lock(Product.class, id);
    p.active = false;
    events.emit("ProductDeactivated", Map.of("id", id, "productId", id));
    return p;
  }

  private void validate(Input q) {
    if (!Set.of("NORMAL", "FRAGILE", "COLD", "HAZARDOUS", "OVERSIZED").contains(q.storageType()))
      throw new IllegalArgumentException("storageType");
    if (!Double.isFinite(q.weight()) || !Double.isFinite(q.length() * q.width() * q.height()))
      throw new IllegalArgumentException("volume");
  }

  public void handle(Event e) {}
}
