package com.personalprojects.pullrequestservice.DTO;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public class ProjectRepoRequestDTO {
    String projectName;
    String repoUrl;
}
