// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"reflect"
	"strconv"
)

// processInterfaceStatus collects [] interface objects with index to preserve ordering as []string{"name/interfaceName[index]=ipAddress"}
// -> ["default/eth-0=[0]1.1.1.1", "default/eth-0[1]=2.2.2.2", "default-2/eth-1[0]=3.3.3.3"]
func processInterfaceStatus(interfaces []reflect.Value) []string {
	var interfaceSlice []string

	for _, iface := range interfaces {
		ifaceData, ok := iface.Interface().(map[string]interface{})
		if !ok {
			continue
		}

		ifaceName, ok := ifaceData["name"].(string)
		if !ok {
			ifaceName = ""
		}

		interfaceStatusName, ok := ifaceData["interfaceName"].(string)
		if !ok {
			interfaceStatusName = ""
		}

		ipAddresses, ok := ifaceData["ipAddresses"].([]interface{})
		if !ok {
			continue
		}

		// build slice of name/interfaceName=ip
		for i, ip := range ipAddresses {
			ipStr, ok := ip.(string)
			if !ok {
				continue
			}
			interfaceSlice = append(interfaceSlice, ifaceName+"/"+interfaceStatusName+"["+strconv.Itoa(i)+"]"+"="+ipStr)
		}
	}

	return interfaceSlice
}
