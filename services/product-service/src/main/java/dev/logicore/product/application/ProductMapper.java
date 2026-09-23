package dev.logicore.product.application;

import dev.logicore.product.domain.Product;
import org.mapstruct.*;

@Mapper(componentModel = "spring")
public interface ProductMapper {
  @BeanMapping(ignoreByDefault = true)
  @Mapping(target = "name", source = "name")
  @Mapping(target = "description", source = "description")
  @Mapping(target = "category", source = "category")
  @Mapping(target = "barcode", source = "barcode")
  void patch(ProductService.Input input, @MappingTarget Product product);
}
