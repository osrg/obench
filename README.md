# What is this?

obench is a benchmark tool for container orchestrator systems (e.g., Kubernetes, Docker InfraKit). It will be designed and implemented for evaluating:

* performance and placement optimality of schedulers
 - Generally speaking, performance (mainly latency) and placement quality of schedulers have a relationship of tradeoff. Sophisticated algorithms can produce good placement decision but require high cost. Naive algorithms can work with low cost but their placements are worse than the sophisticated ones. Understanding the relation is difficult especially in a case of large clusters.
* capability of resource isolation
 - One of the most important motivation of using container orchestratos is increasing utilization of computing resources in clusters. The high utilization is enabled by resource isolation mechanisms of underlying OS e.g., cgroups of Linux. However, it is difficult to determine sufficient resources of each service for achieving performance goals.
* performance of container engines (e.g., dockerd, rkt, ocid)
 - In a realistic environment, many containers will be launched on a single host so performance of container engines needs to be considered (e.g., pulling container images from ra egistry service).

**status is very alpha**: currently, measuring latency of k8s is implemented.
