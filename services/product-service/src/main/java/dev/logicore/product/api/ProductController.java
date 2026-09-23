package dev.logicore.product.api;

import dev.logicore.common.*;
import dev.logicore.product.application.ProductService;
import dev.logicore.product.domain.Product;
import jakarta.validation.Valid;
import java.util.*;
import org.springframework.web.bind.annotation.*;

@RestController
public class ProductController {
  private final ProductService service;
  private final Repository repo;
  private final Commands commands;

  public ProductController(ProductService s, Repository r, Commands c) {
    service = s;
    repo = r;
    commands = c;
  }

  @PostMapping("/products")
  @ResponseStatus(org.springframework.http.HttpStatus.CREATED)
  public Object create(
      @RequestHeader("Idempotency-Key") String key, @Valid @RequestBody ProductService.Input q) {
    return Api.data(commands.run(key, List.of("create", q), () -> service.create(q)));
  }

  @GetMapping("/products")
  public Object list() {
    return Api.data(repo.list(Product.class));
  }

  @GetMapping("/products/{id}")
  public Object get(@PathVariable UUID id) {
    return Api.data(repo.get(Product.class, id));
  }

  @GetMapping("/products/sku/{sku}")
  public Object sku(@PathVariable String sku) {
    var rows = repo.by(Product.class, "sku", sku);
    if (rows.isEmpty()) throw new Fault("NOT_FOUND", "SKU not found", 404);
    return Api.data(rows.getFirst());
  }

  @PutMapping("/products/{id}")
  public Object update(
      @PathVariable UUID id,
      @RequestHeader("Idempotency-Key") String key,
      @Valid @RequestBody ProductService.Input q) {
    return Api.data(commands.run(key, List.of("update", id, q), () -> service.update(id, q)));
  }

  @DeleteMapping("/products/{id}")
  public Object delete(@PathVariable UUID id, @RequestHeader("Idempotency-Key") String key) {
    return Api.data(commands.run(key, List.of("delete", id), () -> service.delete(id)));
  }
}
