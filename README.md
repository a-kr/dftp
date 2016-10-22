# dftp
The goal of this project is to implement a distributed file system with HTTP and FTP interfaces.

Unlike other distributed file systems, `dftp` does not require any kind of file migration to install: it simply projects existing
directory trees on multiple machines into a unified virtual filesystem tree. See "Example" below for details.

The project is still in very early stages of development.

## Roadmap

* [done] Read-only HTTP proxy for local file system.
* Read-only HTTP proxy for a distributed file system.
* Implement read-only FTP interface.
* Implement write operations for HTTP and FTP.

## Example

Suppose you have two machines, `server1` and `server2`, and want to setup a distributed file system
which will contain directory trees from `/storage` and `/data` folders on these machines.

On `server1`:

```
# find /storage
/storage/test.txt
/storage/somedir/a.txt
/storage/somedir/g.gif
# dftp --dfsroot=/storage
2016/10/22 22:08:01 Scanner: local scan finished, 3 file(s) found
2016/10/22 22:08:01 HTTP public interface listening on :7040...
```
On `server2`:
```
# find /data
/data/megafile.tar.gz
/data/somedir/c.jpg
/data/somedir/nested/zzz.bak
# dftp --dfsroot=/data
2016/10/22 22:09:26 Scanner: local scan finished, 3 file(s) found
2016/10/22 22:09:26 HTTP public interface listening on :7040...
```
Now you have a unified distributed file system, the contents of which can be listed using e.g. `/find/` API:
  
```
# curl -s http://server1:7040/find/
test.txt
megafile.tar.gz
somedir/a.txt
somedir/c.jpg
somedir/nested/zzz.bak
somedir/g.gif
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

## HTTP API

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

Returned filenames do not start with "/", but are relative to the root directory of the distributed file system.

