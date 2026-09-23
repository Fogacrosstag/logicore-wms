package dev.logicore.product;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import dev.logicore.common.*;
import dev.logicore.product.application.ProductService;
import org.junit.jupiter.api.Test;

class ProductServiceTest {
  @Test
  void volumeAndEvent() {
    var repo = mock(Repository.class);
    var events = mock(Events.class);
    var service =
        new ProductService(
            repo,
            events,
            org.mapstruct.factory.Mappers.getMapper(
                dev.logicore.product.application.ProductMapper.class));
    var product =
        service.create(
            new ProductService.Input(
                "SKU", "Laptop", "", "electronics", "", 2, 0.4, 0.3, 0.1, "NORMAL"));
    assertEquals(0.012, product.volume, 0.000001);
    assertTrue(product.active);
    verify(repo).add(product);
    verify(events).emit(eq("ProductCreated"), anyMap());
  }

  @Test
  void invalidStorage() {
    var service =
        new ProductService(
            mock(Repository.class),
            mock(Events.class),
            org.mapstruct.factory.Mappers.getMapper(
                dev.logicore.product.application.ProductMapper.class));
    assertThrows(
        IllegalArgumentException.class,
        () ->
            service.create(
                new ProductService.Input("SKU", "Laptop", "", "", "", 1, 1, 1, 1, "INVALID")));
  }
}
