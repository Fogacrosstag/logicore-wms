package dev.logicore.common;

import jakarta.validation.constraints.*;
import java.util.UUID;

public record Line(@NotNull UUID productId, @Min(1) long quantity) {}
