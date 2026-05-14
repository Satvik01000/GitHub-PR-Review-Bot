package com.personalprojects.pullrequestservice.Service.EntrypointService;

import com.personalprojects.pullrequestservice.Service.RedisCacheService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;

@Service
public class EntrypointServiceImple implements EntrypointService {

    private static final Logger logger = LoggerFactory.getLogger(EntrypointServiceImple.class);
    private static final String HMAC_ALGORITHM = "HmacSHA256";
    private final RedisCacheService redisCacheService;

    @Value("${WEBHOOK_SECRET}")
    private String webhookSecret;

    public EntrypointServiceImple(RedisCacheService redisCacheService) {
        this.redisCacheService = redisCacheService;
    }

    @Override
    public void processWebhook(String payload, String event, String signature) {
        logger.info("Received GitHub Webhook Event: {}", event);

        if (!isValidSignature(payload, signature)) {
            throw new SecurityException("Invalid webhook signature");
        }

        logger.info("Signature verified successfully. Processing payload...");

    }

    @Override
    public String saveRepo(String projectName, String repoUrl) {
        redisCacheService.cacheRepoUrl(projectName, repoUrl);
        return "Saved " + projectName + " : " + repoUrl;
    }

    private boolean isValidSignature(String payload, String signature) {
        if (signature == null || !signature.startsWith("sha256=")) {
            return false;
        }

        try {
            String generatedHash = "sha256=" + calculateHmac(payload, webhookSecret);

            return MessageDigest.isEqual(
                    generatedHash.getBytes(StandardCharsets.UTF_8),
                    signature.getBytes(StandardCharsets.UTF_8)
            );
        } catch (Exception e) {
            logger.error("Error calculating HMAC signature", e);
            return false;
        }
    }

    private String calculateHmac(String data, String secret) throws Exception {
        SecretKeySpec secretKeySpec = new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), HMAC_ALGORITHM);
        Mac mac = Mac.getInstance(HMAC_ALGORITHM);
        mac.init(secretKeySpec);

        byte[] rawHmac = mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
        return bytesToHex(rawHmac);
    }

    private String bytesToHex(byte[] bytes) {
        StringBuilder hexString = new StringBuilder(2 * bytes.length);
        for (byte b : bytes) {
            String hex = Integer.toHexString(0xff & b);
            if (hex.length() == 1) {
                hexString.append('0');
            }
            hexString.append(hex);
        }
        return hexString.toString();
    }
}