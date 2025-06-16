This is a go service that listens to changes in resources of pods by listening to the kubernetes api. 
It also has a list of recommendations (provided by external service) so when the resources deviate from the expected amount. It updates in place the pod resources
