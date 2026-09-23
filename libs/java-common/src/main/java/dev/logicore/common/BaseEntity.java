package dev.logicore.common;

import jakarta.persistence.*;
import java.time.Instant;
import java.util.UUID;

@MappedSuperclass
public abstract class BaseEntity {
  @Id public UUID id = UUID.randomUUID();
  public Instant createdAt = Instant.now();
  public Instant updatedAt = Instant.now();
  @Version public long version;

  @PreUpdate
  public void touch() {
    updatedAt = Instant.now();
  }
}
