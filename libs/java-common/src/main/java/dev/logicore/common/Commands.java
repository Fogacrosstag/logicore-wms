package dev.logicore.common;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.security.MessageDigest;
import java.util.*;
import java.util.function.Supplier;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class Commands {
  private final JdbcTemplate db;
  private final ObjectMapper json;

  public Commands(JdbcTemplate db, ObjectMapper json) {
    this.db = db;
    this.json = json;
  }

  @Transactional
  public Object run(String key, Object request, Supplier<Object> action) {
    if (key == null || key.isBlank() || key.length() > 128)
      throw new IllegalArgumentException("Idempotency-Key required");
    try {
      String hash =
          HexFormat.of()
              .formatHex(
                  MessageDigest.getInstance("SHA-256").digest(json.writeValueAsBytes(request)));
      db.queryForList("select pg_advisory_xact_lock(hashtextextended(?,0))", key);
      var rows =
          db.queryForList("select fingerprint,response::text from http_commands where id=?", key);
      if (!rows.isEmpty()) {
        Fault.require(
            rows.getFirst().get("fingerprint").equals(hash),
            "IDEMPOTENCY_CONFLICT",
            "Key used for different request");
        return json.readValue((String) rows.getFirst().get("response"), Object.class);
      }
      Object result = action.get();
      db.update(
          "insert into http_commands(id,fingerprint,response) values(?,?,?::jsonb)",
          key,
          hash,
          json.writeValueAsString(result));
      return result;
    } catch (RuntimeException e) {
      throw e;
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }
}
