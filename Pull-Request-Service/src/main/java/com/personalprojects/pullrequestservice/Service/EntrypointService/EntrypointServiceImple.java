package com.personalprojects.pullrequestservice.Service.EntrypointService;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class EntrypointServiceImple implements EntrypointService {

    private static final Logger logger = LoggerFactory.getLogger(EntrypointServiceImple.class);

    @Override
    public void processWebhook(String payload, String event, String signature) {
        logger.info("Received GitHub Webhook Event: {}", event);
        logger.debug("Signature: {}", signature);
        logger.debug("Payload: {}", payload);

    }
}
