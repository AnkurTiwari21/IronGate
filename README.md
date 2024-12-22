# IronGate - A Dynamic Reverse Proxy for your Docker Containers 🐳

## 📋 Project Overview
This project implements a **Reverse Proxy** with built-in **Load Balancing** and **Rate Limiting** features. The proxy efficiently distributes incoming client requests among backend servers while allowing for dynamic blocking of specific IP addresses.

---

## ✨ Features
1. **Dynamic Routing**  
   - Routes requests to appropriate backend services based on configurable rules or domains.  
   - Enables seamless addition or removal of backend servers without restarting the proxy.  

2. **Load Balancing**  
   - Distributes incoming traffic across multiple backend servers to prevent overload.  
   - Supports round-robin and customizable algorithms for traffic distribution.

3. **Rate Limiting**  
   - Implements IP-based blocking to limit abusive or malicious traffic.  
   - Default setting: Blocks IPs sending more than **100 requests per minute**.  
   - Protects backend servers from denial-of-service or spamming attacks.
     
---

## 🛠️ Technologies Used
- **Programming Language**: Go (Golang)  
- **Web Framework**: Gin for request handling  
- **Load Balancing Logic**: Custom algorithms (resource based + least connections)  
- **Logging**: Logrus for detailed logging  

---

## 💡 V1 Features:

1. ## 🚄 Dynamic Routing:
   - It maintains a Map to store routing information which is dynamically updated on the server.
   - No need to restart server in case a new resource is updated.
     
2. ## ⚖️ Load Balancing:
   1. **Scale Up 🚀**
     
       - I have tried to implement a combination of Resource-based and Least connection to achieve load balancing.

     
       - ```go
         func (r *ReverseProxy) FindMatch(url string, imageMapping *mapping.ImageMapping) string {
          // using a resource based alocation method
	      // find avg cpu usage for that domain
	      // if avg cpu usage > 80 --> spin up another conatiner and route traffic there
	      // otherwise route traffic to coatiner with lowest cpu usage
	      // return the container id

	      avgCPUUsage := float64(0)
	      var containerWithMinCPUUsage string
	      minCPUUsage := float64(1000)

	      for _, conatiner := range r.Routes[url] {
		    cpuUsage, err := containerhandler.MonitorContainerWithID(conatiner)
		    if err != nil {
			    logrus.Errorf("error getting stats for container : %s | err ", conatiner)
			    logrus.Error("error is ", err)
		    } else {
			    avgCPUUsage += (cpuUsage)
			    if cpuUsage <= minCPUUsage {
				    if cpuUsage == minCPUUsage {
					    if containerWithMinCPUUsage != "" && r.RequestPerContainerPerSecond[conatiner] <= r.RequestPerContainerPerSecond[containerWithMinCPUUsage] {
						    containerWithMinCPUUsage = conatiner
					    }
				    } else {
					    minCPUUsage = cpuUsage
					    containerWithMinCPUUsage = conatiner
				    }
			    }

		    }
	      }
     
	      avgCPUUsage = avgCPUUsage / float64(len(r.Routes[url]))
	      if avgCPUUsage > 80 {
		    conatinerUUId := uuid.New()
		    containerhandler.RunContainerFromImageInBackground(imageMapping.Mapping[url], "ankur-net", conatinerUUId.String())
		    r.Add(url, conatinerUUId.String())
		    r.Mu.Lock()
		    r.RequestPerContainerPerSecond[conatinerUUId.String()] += 1
		    r.Mu.Unlock()
		    return conatinerUUId.String()
	      }
	      r.Mu.Lock()
	      r.RequestPerContainerPerSecond[containerWithMinCPUUsage] += 1
	      r.Mu.Unlock()
	      return containerWithMinCPUUsage
          }
         ```
    - Above function aims to find the find the server with **minimum cpu utiliztion** and route the traffic there.
    - If the average cpu utilizing across all servers is **>80%** it will spin up another server.
    - Priority will be given to **Minimum CPU utilizing server**, if we have a tie then we will route traffic to the server with least connections
    - Spinning Up server is done gracefully
      
    - **Docker Logs**:
        - <img width="1205" alt="Screenshot 2024-12-22 at 7 08 16 PM" src="https://github.com/user-attachments/assets/1cf15c13-c474-4b10-8c1a-5e5354255843" />
        - Above is a snapshot of conatiners started to handle the load running the same server.


    2. **Scale down** 📉

       - ```go
             //perform cleanup of unused container here
	        //using go routine to do process async
	        //using time.NewTicker to do in a constant interval
	        //default time for checking is 5min
	        go func() {
		      ticker := time.NewTicker(5 * time.Minute)
		      defer ticker.Stop()
		      for {
			    select {
			    case <-ticker.C:
				logrus.Info("Checking for unused containers....")
				for url, containers := range rp.Routes {
					containerToBeRemoved, avgCPUUsage := containerhandler.ScaleDownContainers(containers)
					if containerToBeRemoved != "" {
						err := containerhandler.StopContainerByIdOrName(containerToBeRemoved)
						logrus.WithFields(logrus.Fields{
							"container": containerToBeRemoved,
							"avg_cpu":   avgCPUUsage,
						}).Info("Scaling down container due to low average CPU utilization")

						if err != nil {
							logrus.Error("Error stoping container | err ", err)
						}
					}
					rp.RemoveContainer(url, containerToBeRemoved)
				}
			}
		  }
	      }()
         ```

         - Above Process is responsible to check for unused containers and remove them
         - This process runs every 5 min (default).
         - Logs:
             - <img width="1126" alt="Screenshot 2024-12-22 at 7 01 56 PM" src="https://github.com/user-attachments/assets/168e83c1-44df-4e27-b3be-ab660738e492" />
           
 3. ## ⛔ Rate Limiting - 
    - Used **Redis** for caching the IP.
    - If **>100 request/min/IP** then we block the IP for 5 minutes (Default).
    - <img width="882" alt="Screenshot 2024-12-22 at 6 59 42 PM" src="https://github.com/user-attachments/assets/5afcc578-2512-4043-bd61-ede2ca02b419" />

## 🧪 Performance Testing with JMeter

### Test Configuration:
- **Tool Used**: Apache JMeter  
- **Users**: 100 users  
- **Ramp-Up Period**: 30 seconds  
- **Duration**: Continuous requests for 60 seconds  

### Results:
- **Request Throughput**: ~125 requests/second  
- **Average Latency**: ~300ms  
- **Error Rate**: 0% (No failed requests)  
- **CPU Usage**: Balanced across backend servers due to load balancing.  

**Observation**:  
- The proxy handled all 100 users seamlessly, distributing traffic to multiple backend servers without overload.  
- Rate limiting successfully blocked excessive requests from specific IPs as expected.  

---
