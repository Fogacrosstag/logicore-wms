package dev.logicore.common;

import org.apache.kafka.common.TopicPartition;
import org.springframework.context.annotation.*;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.listener.*;
import org.springframework.util.backoff.FixedBackOff;

@Configuration
public class KafkaConfig {
  @Bean
  DefaultErrorHandler kafkaErrorHandler(KafkaTemplate<String, String> template) {
    var recoverer =
        new DeadLetterPublishingRecoverer(
            template, (record, error) -> new TopicPartition("wms.events.DLT", record.partition()));
    recoverer.setFailIfSendResultIsError(true);
    return new DefaultErrorHandler(recoverer, new FixedBackOff(1000L, 4L));
  }
}
