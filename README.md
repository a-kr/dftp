# dftp
The goal of this project is to implement a distributed file system with HTTP and FTP interfaces.

Unlike other distributed file systems, `dftp` does not require any kind of file migration to install: it simply projects existing
directory trees on multiple machines into a unified virtual filesystem tree. See "Example" below for details.

The project is still in very early stages of development.

## Roadmap

* [done] Read-only HTTP proxy for local file system.
* [done] Read-only HTTP proxy for a distributed file system.
* [done] Implement read-only FTP interface.
* [done] Implement peer discovery based on multicast UDP messages.
* Implement periodic updates and local filesystem changes monitoring.
* Implement write operations for HTTP and FTP.

## Example

Suppose you have two machines, `server1` and `server2`, and want to setup a distributed file system
which will contain directory trees from `/storage` and `/data` folders on these machines.

Start dftp on `server1`:

```
# find /storage
/storage/test.txt
/storage/somedir/a.txt
/storage/somedir/g.gif
# dftp --dfsroot=/storage
2016/10/22 22:08:01 Scanner: local scan finished, 3 file(s) found
2016/10/22 22:08:01 HTTP public interface listening on :7040...
2016/10/22 22:08:01 HTTP mgmt interface listening on :7041...
2016/10/22 22:08:01 FTP public interface listening on :2121...
```
...And on `server2`:
```
# find /data
/data/megafile.tar.gz
/data/somedir/c.jpg
/data/somedir/nested/zzz.bak
# dftp --dfsroot=/data
2016/10/22 22:09:26 Scanner: local scan finished, 3 file(s) found
2016/10/22 22:09:26 HTTP public interface listening on :7040...
2016/10/22 22:09:26 HTTP mgmt interface listening on :7041...
2016/10/22 22:09:26 FTP public interface listening on :2121...
```

Nodes will automatically discover each other via multicast. Alternatively, you can manually ask one node to join the other:
```
# curl -d 'peer=server2:7041' http://server1:7041/join/
```

Now you have a unified distributed file system, the contents of which can be listed using e.g. `/find/` API:

```
# curl -s http://server1:7040/find/
/test.txt
/megafile.tar.gz
/somedir/a.txt
/somedir/c.jpg
/somedir/nested/zzz.bak
/somedir/g.gif
```

Querying `server2` will yield the same response.

Contents of a file can be retrieved over HTTP by adding `/fs/` to full file path:

```
# curl -s http://server1:7040/fs/test.txt
This is a text file
```

## Installation

You will need Go 1.5+ and GNU Make to build `dftp`.

```
git clone https://github.com/a-kr/dftp
cd dftp
make
bin/dftp --help
```

## Command line options

```
Usage of bin/dftp:
  -cluster-name string
        cluster name (change it to allow multiple separate clusters work with same multicast discovery address) (default "dftp")
  -dfsmount string
        path inside DFS where local tree will be mounted (not necessarily unique path)
  -dfsroot string
        local directory corresponding to local DFS root
  -ftp-listen string
        host:port for public FTP interface to listen on (default ":2121")
  -http-listen string
        host:port for public HTTP interface to listen on (default ":7040")
  -http-mgmt-listen string
        host:port for private cluster management HTTP interface to listen on (default ":7041")
  -multicast-discovery-addr string
        host:port for multicast peer discovery address (default "224.0.0.9:7041")
  -node-name string
        node name to use instead of hostname

```

## HTTP API

Public HTTP API is available by default on port `:7040`.

* `GET /`

Displays simple greeting page.

* `GET /fs/`

Displays nginx-like HTML directory listing for the root directory of the distributed file system.

* `GET /fs/<path>`

If `path` points to a directory, displays nginx-like directory listing for this directory.
Otherwise, serves the file contents as HTTP response, guessing Content-Type from filename extension.

* `GET /find/`

Returns complete list of full filenames for every file in the distributed file system, much like Unix `find` command does,
as a `text/plain` newline-separated response. For large filesystems this command may take quite a while to execute.

Returned filenames start with "/" and are relative to the root directory of the distributed file system.


## Peer discovery

`dftp` supports automatic peer discovery via multicast, and manual discovery using `POST /join/` HTTP requests.

Multicast discovery is enabled by default. Each node periodically (every minute) announces its name, management API address:port, and cluster name. When a node receives such announcement from a previously-unmet peer, the node _greets_ the peer (`POST /cluster/` request to peer's management API, which is detailed below).

Multicast address and port can be specified with `--multicast-discovery-addr` command line option. If for some reason you need to operate multiple separate clusters on the same address and port, you must specify a different `--cluster-name` for each cluster.

Multicast is not used for anything other than peer discovery.

## Internal cluster communication

Peer to peer communication happens over HTTP on port `:7041`.

* Every node speaks to every other node directly. No 'masters' are elected.
* A new node joins cluster by _greeting_ (`POST /cluster/`) any known node. The new node receives a list of cluster nodes in return. The new node then _greets_ every other node in the list and requests _full updates_ from each of them.
* An _update_ is a list of files (and their attributes) local to the sender node. A _full update_ contains all files; by contrast, an incremental update contains only some of them (e.g. files which have been changed since last full update).
* A node is responsible for pushing updates to every other node. These updates are not propagated further.
* Every node stores a complete tree representation of the distributed file system, and maintains it by both receiving updates from other nodes and scanning its own local filesystem.
* [TODO] Every node sends incremental updates upon observing changes in the local filesystem. Every node also sends full updates periodically (every hour by default).
* [TODO] Upon receiving a _full update_, a node prunes all files which were marked to belong to sender node, but are not contained in the full update. Thus file deletion is handled.
* [TODO] Every node periodically pings every other node with `POST /cluster/` request without requesting a full update. Nodes which do not respond to such request are removed from cluster, along with all the files they own.
* If several nodes contain a file with the same path locally, the file will be considered belonging to that node which has sent the more recent update containing this file. File modification time and other attributes are not considered in conflict resolution.
* The described distributed system is _eventually consistent_ with regard to file information.

Description of the cluster management API follows.

* `POST /join/`

Used to bootstrap peer discovery process for new nodes if for some reason multicast discovery is not enough. Required form parameter is `peer`, which must contain an address of any other cluster node's management API endpoint in the form of `<host>:<port>` (where port is usually 7041). Upon receiving this command, the node sends a _greeting_ to specified node.

Example:

```
curl -d 'peer=server2.org:7041' http://server1.org:7041/join/
```

* `GET /cluster/`

Returns information about the node (name, public API address, management API address), and a list of other cluster nodes, with the same attributes.

* `POST /cluster/`

Sends a _greeting_, asking the node to update information on the caller. Form parameters are:

  1. `name`: name of the calling node;
  2. `public-addr`: address of public HTTP API endpoint, in the form of `<host>:<port>`, where `<host>` may be empty;
  3. `mgmt-addr`: address of management HTTP API endpoint;
  4. `request-full-update`, optional. If equals `true`, the node must push a _full update_ to the calling node, by sending a `POST /update/` request asynchronously after processing the greeting request.

Response is the same as for `GET /cluster/`.

* `POST /update/`

Sends an _update_, asking the node to amend its information about files and attributes. POST body must be a JSON document:
```
{
  "SenderNodeName": "server1",
  "Full": true,
  "UpdateTime": 1477224426,
  "Files": [
    {
      "Deletion": false,
      "FullName": "somefolder",
      "Basename": "somefolder",
      "Dir": true,
      "LastModified": 1476551310,
      "LastInfoUpdated": 1477224426,
      "SizeInBytes": 0,
      "FileMode": 2147484141,
      "OwnerNode": "server1"
    },
    {
      "Deletion": false,
      "FullName": "somefolder/test.txt",
      "Basename": "test.txt",
      "Dir": false,
      "LastModified": 1476551310,
      "LastInfoUpdated": 1477224426,
      "SizeInBytes": 123,
      "FileMode": 2147484141,
      "OwnerNode": "server1"
    },
    ...
  ]
}
```

The order of files is arbitrary. Directories may be skipped; they are created on the fly upon encountering any files contained within. If `"Deletion"` is true, the node treats the item as file removal notification and makes the file unavailable for reading.

Upon successful parsing of the update request, the node responds with simple "ok" and starts applying updates to its own copy of filesystem tree asynchronously.
