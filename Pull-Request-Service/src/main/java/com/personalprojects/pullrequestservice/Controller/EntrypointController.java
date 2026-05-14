package com.personalprojects.pullrequestservice.Controller;

import com.personalprojects.pullrequestservice.DTO.ProjectRepoRequestDTO;
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
public class EntrypointController {

    private final EntrypointService entrypointService;

    @Autowired
    public EntrypointController(EntrypointService entrypointService) {
        this.entrypointService = entrypointService;
    }

    @PostMapping("/repo")
    public ResponseEntity<String> saveRepo(@RequestBody ProjectRepoRequestDTO projectRepoRequestDTO){
        return ResponseEntity.ok(entrypointService.saveRepo(projectRepoRequestDTO.getProjectName(), projectRepoRequestDTO.getRepoUrl()));
    }

    @PostMapping("/webhook")
    public ResponseEntity<Void> onWebhook(
            @RequestHeader(value = "X-GitHub-Event", required = true) String event,
            @RequestHeader(value = "X-Hub-Signature-256", required = true) String signature,
            @RequestBody String payload) {
        entrypointService.processWebhook(payload, event, signature);
        return ResponseEntity.ok().build();
    }
}