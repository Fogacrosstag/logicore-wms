package dev.logicore.shipment.application;

import dev.logicore.common.Fault;

public final class StateMachine {
  private StateMachine() {}

  public static void require(String actual, String expected) {
    Fault.require(
        expected.equals(actual), "INVALID_STATE", "Expected " + expected + ", got " + actual);
  }
}
