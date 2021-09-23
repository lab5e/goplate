package template

// After much pondering "let's roll our own template engine" was the conclusion
// The go template engine in text/template is nice but has way more features
// than we need (or even *should expose*) but the syntax is okay'ish. The
// main problem with this is that the template syntax and field names will be
// spelled differently from the JSON exposed via the API. The output from
// the service looks roughly like this:
//
//      {
//      	"type": "data",
//      	"device": {
//      		"deviceId": "566ecm37cmcahh",
//      		"collectionId": "566ecm37cmcahj",
//      		"imsi": "242016000661854",
//      		"imei": "352656100987596",
//      		"tags": {
//      			"3gpp-ms-timezone": "8001",
//      			"3gpp-user-location-info": "8242f21076c142f210010bcc01",
//      			"name": "Thingy91 GW (Lab5e)",
//      			"radius-allocated-at": "2021-09-07T12:37:09Z",
//      			"radius-ip-address": "10.0.0.52"
//      		},
//      		"network": {
//      			"allocatedIp": "10.0.0.52",
//      			"allocatedAt": "1631018229880"
//      		},
//      		"firmware": {
//      			"currentFirmwareId": "0",
//      			"targetFirmwareId": "0",
//      			"firmwareVersion": "",
//      			"serialNumber": "",
//      			"modelNumber": "",
//      			"manufacturer": "",
//      			"state": "Current",
//      			"stateMessage": ""
//      		},
//      		"metadata": {
//      			"simOperator": {
//      				"mcc": 242,
//      				"mnc": 1,
//      				"country": "Norway",
//      				"network": "Telenor"
//      			}
//      		}
//      	},
//      	"payload": "CM/PjMIDEAgw6gE4z44GQOsDaOL5/////////wFwj///////////AXi7jAGFAQAAAECNAQAAWEKVAatS9kKiAQYAJQCMNJ6qAQYAJwApd56wAQM=",
//      	"received": "1631023436264",
//      	"transport": "udp",
//      	"udpMetaData": {
//      		"localPort": 31415,
//      		"remotePort": 61009
//      	},
//      	"messageId": "16a28f2c7ae619970000000000000004"
//      }
//
// The templates should map 1:1 to the field names and structure for since it
// is much easier to understand and write templates just by looking around in
// the documentation.
// In addition to the device and message we'll include a collection object that
// can be used.
//
// The templates use {{ and }} as the separators just like the Go template engine
//
