package dev.logicore.shipment;

import static org.junit.jupiter.api.Assertions.*;

import dev.logicore.common.Fault;
import dev.logicore.shipment.application.StateMachine;
import org.junit.jupiter.api.Test;

class StateMachineTest {
  @Test
  void transitionGuards() {
    assertDoesNotThrow(() -> StateMachine.require("PICKING", "PICKING"));
    assertThrows(Fault.class, () -> StateMachine.require("SHIPPED", "PICKING"));
  }
}
