package dev.logicore.common;

import java.time.Instant;
import java.util.*;
import org.slf4j.MDC;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.http.*;
import org.springframework.web.bind.annotation.*;

@RestControllerAdvice
public class Api {
  public static Map<String, Object> data(Object value) {
    return Map.of("data", value, "timestamp", Instant.now());
  }

  @ExceptionHandler(Fault.class)
  ResponseEntity<?> fault(Fault e) {
    return error(e.status, e.code, e.getMessage());
  }

  @ExceptionHandler({
    org.springframework.web.bind.MethodArgumentNotValidException.class,
    org.springframework.http.converter.HttpMessageNotReadableException.class,
    IllegalArgumentException.class,
    org.springframework.web.bind.MissingRequestHeaderException.class,
    org.springframework.web.method.annotation.MethodArgumentTypeMismatchException.class
  })
  ResponseEntity<?> invalid(Exception e) {
    return error(
        400, "VALIDATION_ERROR", "Invalid request fields, UUID, or missing Idempotency-Key");
  }

  @ExceptionHandler(DataIntegrityViolationException.class)
  ResponseEntity<?> conflict(Exception e) {
    return error(409, "CONFLICT", "Unique key or database constraint violated");
  }

  @ExceptionHandler(Exception.class)
  ResponseEntity<?> unexpected(Exception e) {
    org.slf4j.LoggerFactory.getLogger(Api.class).error("Request failed", e);
    return error(500, "INTERNAL_ERROR", "Unexpected server error");
  }

  private ResponseEntity<?> error(int status, String code, String message) {
    return ResponseEntity.status(status)
        .body(
            Map.of(
                "code",
                code,
                "message",
                message,
                "details",
                Map.of(),
                "timestamp",
                Instant.now(),
                "traceId",
                Objects.toString(MDC.get("traceId"), "")));
  }
}
