package dev.logicore.common;

import jakarta.persistence.*;
import java.util.*;
import org.springframework.stereotype.Component;

@Component
public class Repository {
  @PersistenceContext private EntityManager em;

  public <T> T get(Class<T> type, UUID id) {
    T e = em.find(type, id);
    if (e == null) throw new Fault("NOT_FOUND", type.getSimpleName() + " not found", 404);
    return e;
  }

  public <T> T lock(Class<T> type, UUID id) {
    T e = em.find(type, id, LockModeType.PESSIMISTIC_WRITE);
    if (e == null) throw new Fault("NOT_FOUND", type.getSimpleName() + " not found", 404);
    return e;
  }

  public <T> T add(T entity) {
    em.persist(entity);
    em.flush();
    return entity;
  }

  public <T> List<T> list(Class<T> type) {
    return em.createQuery("from " + type.getSimpleName() + " order by createdAt", type)
        .setMaxResults(500)
        .getResultList();
  }

  public <T> List<T> by(Class<T> type, String field, Object value) {
    return em.createQuery(
            "from " + type.getSimpleName() + " where " + field + "=:value order by createdAt", type)
        .setParameter("value", value)
        .setMaxResults(500)
        .getResultList();
  }
}
