
### Kafka Events:

* collector_started      - Will start streaming initial state.
* collector_initialized  - Done sending initial state, will send new deltas. 
* resource_add           - Example: {uid:123 Properties:["a":"aaa", "b":"bbb", "c":"ccc"]}
* resource_update        - Example: {uid:123 Properties:["b": nil, "c":"changedVal", "d":"newProp"]}
* resource_delete        - Example: {uid:123}