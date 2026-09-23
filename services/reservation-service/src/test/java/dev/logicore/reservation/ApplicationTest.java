package dev.logicore.reservation;

import static org.junit.jupiter.api.Assertions.*;

import dev.logicore.common.Events;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

@SpringBootTest(properties = {"spring.kafka.listener.auto-startup=false"})
@Testcontainers(disabledWithoutDocker = true)
class ApplicationTest {
  @Container
  static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:16.9-alpine");

  @DynamicPropertySource
  static void properties(DynamicPropertyRegistry r) {
    r.add("DB_URL", postgres::getJdbcUrl);
    r.add("POSTGRES_USER", postgres::getUsername);
    r.add("POSTGRES_PASSWORD", postgres::getPassword);
  }

  @MockitoBean Events events;
  @Autowired JdbcTemplate db;

  @Test
  void migrationsAndHibernateAgree() {
    assertEquals(0, db.queryForObject("select count(*) from reservations", Integer.class));
    assertEquals(0, db.queryForObject("select count(*) from outbox_events", Integer.class));
  }
}
