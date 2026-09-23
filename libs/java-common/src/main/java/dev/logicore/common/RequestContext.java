package dev.logicore.common;

import jakarta.servlet.*;
import jakarta.servlet.http.*;
import java.io.IOException;
import java.util.UUID;
import org.slf4j.MDC;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

@Component
public class RequestContext extends OncePerRequestFilter {
  protected void doFilterInternal(HttpServletRequest q, HttpServletResponse p, FilterChain chain)
      throws ServletException, IOException {
    String c = q.getHeader("X-Correlation-ID");
    if (c == null || !c.matches("[a-zA-Z0-9-]{1,80}")) c = UUID.randomUUID().toString();
    MDC.put("correlationId", c);
    String t = q.getHeader("traceparent");
    if (t != null && t.matches("[a-f0-9-]{55}")) MDC.put("traceparent", t);
    p.setHeader("X-Correlation-ID", c);
    try {
      chain.doFilter(q, p);
    } finally {
      MDC.remove("correlationId");
      MDC.remove("traceparent");
    }
  }
}
