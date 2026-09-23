package dev.logicore.reservation;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication(scanBasePackages = {"dev.logicore.reservation", "dev.logicore.common"})
@EntityScan("dev.logicore.reservation")
@EnableScheduling
public class Application {
  public static void main(String[] args) {
    SpringApplication.run(Application.class, args);
  }
}
