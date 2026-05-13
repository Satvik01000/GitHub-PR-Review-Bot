package com.personalprojects.pullrequestservice.Controller;

import com.personalprojects.pullrequestservice.Service.EntrypointService.EntrypointService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
public class Entrypoint {

    private final EntrypointService entrypointService;

    @Autowired
    public Entrypoint(EntrypointService entrypointService) {
        this.entrypointService = entrypointService;
    }

    @PostMapping("/webhook")
    public ResponseEntity<Void> onWebhook(
            @RequestHeader(value = "X-GitHub-Event", required = false) String event,
            @RequestHeader(value = "X-Hub-Signature-256", required = false) String signature,
            @RequestBody String payload) {
        entrypointService.processWebhook(payload, event, signature);
        return ResponseEntity.ok().build();
    }
}