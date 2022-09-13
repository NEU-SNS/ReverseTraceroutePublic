# Ranking service

This is the documentation for the Ranking Service component of Reverse Traceroute.

Ranking Service is in charge of providing vantage points (VP) to use in the next spoofed RR round for a target destination. Before Ranking Service, revtr used to randomly pick a VP from the list of available and active VPs. Ranking Service uses network heuristics under the hood to provide VPs that are most likely to reach the target within 8 hops, effectively making Reverse Traceroute run faster.

Ranking Service has two essential components: 
- The server, which provides an API endpoint to ask for VPs. See API for more details.
- The ranking generator, which does periodic measurements and generates fresh tables that rank the VP sites which contain Reverse Traceroute VPs.

## API

### Overview
 
Ranking Service offers getVPs(d, k, r, m) where:
- `d`: Destination IP.
- `k`: Number of requested VPs.
- `r`: Ranking technique (this can currently be "TK" for Top-K ranking, or "IC" for Ingress Cover ranking).
- `m`: List of measurements issued so far by the user.

The returned object is a list of Sources. A Source is a tuple representing a vantage point. It contains the hostname, IP and site name for the vantage point. This is a gRPC API, which means that Google's protobuffers are used for the requests and responses of the API. Since protobuffers are language agnostic, other components of Reverse Traceroute (all written in Go) can easily interact with Ranking Service (which is written in Python).

### Instructions

Currently, the service runs on Walter, under the user egurmeri. `rankingservice_cron.sh` runs every minute, making sure that `rankingservice_server.py` is up and running. The latter is the component that hangles the API call. If you carry the code to another account, make sure to pip install necessary dependencies (a Docker container sounds like a nice TODO).

Below is a schema illustrating the code logic behind the API.

![test](https://github.com/NEU-SNS/ReverseTraceroute/blob/egurmeri/rankingservice/images/RankingServiceAPI.png?raw=true)

### How does this fit in Reverse Traceroute?

1. We need to keep track of executed measurements (i.e. `m`), which wasn't a feature in Reverse Traceroute. Now, each Reverse Traceroute session has a `RRHop2IssuedProbes` map, which is a destination-->List[measurementID].

2. The regular call to get Vantage Points (i.e. random selection from active nodes) needs to change with the API call. The affected methods are `InitializeRRVPs()` and `GetRRVPs()`. Both functions in the codebase are followed by their new counterpart that uses the API (e.g. `InitializeRRVPs_new()`). See TODO list to get these new methods up and running. `GetRankedSpoofers()` is the method that calls our gRPC API.



#### Top-K ranking (default)

As a first step, Ranking Service fetches all measurements within the past 10 minutes to the destination `d` from Reverse Traceroute's measurements database (we call those "cached" measurements). If all such measurements are unresponsive (and if there is a significant amount of measurements), Ranking Service assumes that the destination is unresponsive, and returns an empty set of Sources.

Second, Ranking Service fetches the next `k` vantage points with the highest ranking scores. The scores are determined using the set cover algorithm. We start with a set of destinations (typically a hitlist of actives IPs from all BGP routable prefixes in the world), issue Record Route (RR) option enabled pings to all destinations from all VPs, and run a greedy set cover algorithm, where a VP covers a destination if the RR trace contains the destination IP.

Third, we filter out any selected VPs that fall in at least one of the following categories:
- inactive: quarantined by vpservice, another Reverse Traceroute component.
- used: `m` contains a measurement from the VP in question.
- unresponsive: cached measurements show that lost measurements from the VP's site.

If, after filtering out, the number N of VPs left is greater or equal to `k`, Ranking Service returns those. Otherwise, it loops back to fetching the next `k` VPs with the highest scores and applying the aforementioned filtering.

#### Ingress Cover ranking

TODO.


## Ranking generator

Let's have a look at the tables inside the `ranking` database (in SkylerDB)

*blacklist*
![test](https://github.com/NEU-SNS/ReverseTraceroute/blob/egurmeri/rankingservice/images/blacklist.png?raw=true)

*hitlist*
![test](https://github.com/NEU-SNS/ReverseTraceroute/blob/egurmeri/rankingservice/images/hitlist.png?raw=true)

*targets*
![test](https://github.com/NEU-SNS/ReverseTraceroute/blob/egurmeri/rankingservice/images/targets.png?raw=true)


The functionality for each of steps of the ranking generator are mostly written and independently tested with dummy data. A todo is to finish the parts that still need work and then bundle things in a cron, so that we have fresh new target IPs say every week.

### 0. Using an updated hitlist.

This step cannot run on Walter, as it requires a python version that is not on Walter. A very useful TODO is to update python properly so that this step can also be done on Walter.

There is no automted way to download hitlists at the moment, so downloading it and placing it in the project has to be manually done. You should create a  `hitlists` folder in the `ranking_periodic` section of your local repo copy, which should contain the following:

![test](https://github.com/NEU-SNS/ReverseTraceroute/blob/egurmeri/rankingservice/images/hitlists_content.png?raw=true)

For the symlinks:
`latest` should point to `internet_address_hitlist_it86w-<timestamp>/internet_address_hitlist_it86w_<timestamp>.fsdb`
`hitlists` should point to `./hitlists`

Running setup_periodic.sh should parse the hitlist and dump a csv file in `csv/destinations`. The `latest` symlink in that folder will point to that specific file. The last step of `setup_periodic.sh` sends that csv file to walter. You will have to change the ssh key and account names with your respective credentials. Once the file is uploaded, head over to the `ranking_periodic` folder on Walter and run:
```
./load_hitlist_data_to_db.py
```
This will parse the csv and insert the new content in the `hitlist` table on Skyler's ranking DB.

### 1. Define how to select destinations from one round to another.

Every round, some destinations fail, some destinations are consistently unreachable via RR -- long story short, we want to use observations from the previous round of measurements to improve the list of destinations to probe in the next round. `update_ranking_targets.py` in `rankingservice` takes care of this. Run the following:
```
python update_ranking_targets.py <measurement_round_label_name> 
```
This script does two things. First, it does the post-mortem for the previous round. That entails looking at each probe destination and checking if it was unresponsive. If so, the IP is set to NULL in the row in the `targets` table corresponding to the BGP prefix of that destination and the IP is added to the `blacklist` table.

Second, it follows the algorithm described [here](https://docs.google.com/document/d/1qB8-e8WqwNupx7YhylhSfrkmlePRlLEKbIuaZjLZfuA/edit#bookmark=id.jc88b3pg6818) here to find potentially better destinations for each BGP prefix. If you cannot access the document, contact Ethan/Dave.

### 2. Issuing measurement rounds every week
Various API calls have been created to issue record routes from MLab VPs using the underlying infrastructure built for Reverse Traceroute.
You can find these API functions in `revtr/v1api/v1api.go`. Because during the creation of this system, MLab deployment was put on hold due to a long standing issue, we couldn't productionize this part. A TODO is to write a system that calls those API endpoints so that each VP in the list of VPs available to Reverse Traceroute issues a measurement to each IP in `targets` with an appropriate measurement round label name.

`recordroute` is the API call that you will likely need--it takes as argument as list of destinations and issues pings to each destination from each of the sources available to Revtr. Right now, the label is set to be YYYY-MM-DD at time of measurement. This might or might not be a good pattern, as some rounds can last more than a day. A better option (TODO) is to update the code to take the label name as an argument in the API call.

### 3. Processing the measurements.
 
 This step entails: reading measurements of the latest round, building a ranking using a ranking technique (e.g. Top-K) and inserting ranked VPs into the ranking table: `process_measurements.py` takes care of all of this. See code for details on how to use it.


## TODO list.

- [ ] Docker'ize Ranking Service (mostly for cohesion with the rest of the system structure and to automate/simplify steps such as library dependencies
- [ ] `GetRankedSpoofers()` has a hardcoded IP:port variable. A more robust solution is to read the IP that corresponds to docker0.
- [ ] Change `InitializeRRVPs()` and `GetRRVPs()` with their "*_new" counterpart and test, once MLab deployment is successful and Reverse Traceroute can be truly tested.
- [ ] Perform a full Ranking Generator test round, once MLab deployment is successful.
- [ ] Step 2 of ranking generator.
- [ ] productionize ranking generator by bundling separate parts in a cron that runs every week.
