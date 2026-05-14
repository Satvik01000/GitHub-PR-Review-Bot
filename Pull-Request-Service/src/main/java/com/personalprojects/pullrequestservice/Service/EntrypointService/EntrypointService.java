package com.personalprojects.pullrequestservice.Service.EntrypointService;

public interface EntrypointService {
    void processWebhook(String payload, String event, String signature);
    String saveRepo(String projectName, String repoUrl);
}
