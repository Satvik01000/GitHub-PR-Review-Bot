package com.personalprojects.pullrequestservice.Service;

import java.time.Duration;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class RedisCacheService {

    private final RedisTemplate<String, String> redisTemplate;

    private static final Duration CACHE_TTL = Duration.ofHours(1);

    public RedisCacheService(RedisTemplate<String, String> redisTemplate) {
        this.redisTemplate = redisTemplate;
    }

    private String getKey(String projectName) {
        return "project_name:" + projectName;
    }

    public void cacheRepoUrl(String projectName, String repoUrl) {
        String key = getKey(projectName);
        redisTemplate.opsForValue().set(key, repoUrl, CACHE_TTL);
    }

    public String getRepoUrl(String projectName) {
        String key = getKey(projectName);
        return redisTemplate.opsForValue().get(key);
    }

    public void removeRepoUrl(String projectName) {
        String key = getKey(projectName);
        redisTemplate.delete(key);
    }
}
