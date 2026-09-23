package dev.logicore.common;

public class Fault extends RuntimeException {
  public final String code;
  public final int status;

  public Fault(String code, String message, int status) {
    super(message);
    this.code = code;
    this.status = status;
  }

  public static void require(boolean ok, String code, String message) {
    if (!ok) throw new Fault(code, message, 409);
  }
}
