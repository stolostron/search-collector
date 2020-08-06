# Document edge creation

Document how we create relationships between resources.

### Common
  - **(\*)-[OWNED_BY]->(\*)**
    - Extract owner references from the object's metadata.

### Application

- **(Application)-[CONTAINS]->(Subscriptions)**
  - Use the annotation `apps.open-cluster-management.io/subscriptions` to link subscriptions associated to the application.

- **(Application)-[CONTAINS]->(Deployable)**
  - Use the annotation `apps.open-cluster-management.io/deployables` to link deployables associated to the application.

- ?? How do we use `_hostingApplication` ??


### Channel

- **(Channel)-[USES]->(ConfigMap)** OR **(Channel)-[USES]->(Secret)**
  - Extract from spec.

- **(Channel)-[DEPLOYS]->(Deployable)**
  - If channel type is a helm repo, extract from spec.

### Subscription

- **(Subscription)-[TO]->(Channel)**
  - Extract from `Spec.Channel`

- **(Subscription)-[REFERS_TO]->(PlacementRule)**
  - Extract from `Spec.Placement.PlacementRef.Name`

- **(Subscription)-[SUBSCRIBES_TO]->(Deployable)**
  - Use the annotation `apps.open-cluster-management.io/deployables` on the subscription.

- **(\*)-[DEPLOYED_BY]->(Subscription)**
  - Use the annotation `_hostingSubscription` on any resource to link to the subscription that created the resource.


### Deployable (AppDeployable)

- **(Deployable)-[PROMOTED_TO]-(Channel)**
  - Extract from `Spec.Channels`

- **(Deployable)-[REFERS_TO]-(PlacementRule)**
  - Extract from `Spec.Placement.PlacementRef.Name`

- ?? How do we use `_hostingDeployable` ??