package dev.logicore.common;

import java.time.Instant;
import java.util.*;

public record Event(
    UUID eventId,
    String eventType,
    int eventVersion,
    Instant timestamp,
    String source,
    String correlationId,
    String traceparent,
    Map<String, Object> payload) {}
