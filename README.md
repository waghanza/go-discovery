## Go lang discovery project

This API has 2 **endpoints** :
- /ping
- /repos

## Architecture

#### Background task

> `cron.go`

This is a background task that fetch (from **github**) and insert into a SQLite database some repositories info

###### Usage

```sh
go run cron.md $TOKEN
```

`$TOKEN` is github access token :information_source: Anonymous call could fac e github API rate limiting

#### API

> `/ping`

returns a json
```json
{
    "status": "pong"
}
```

this could be helpful for status check

> `/repos`

get some data about repos
```json
[
    {
        "archived": false,
        "id": 71,
        "name": "ruby-on-rails-tmbundle",
        "stars":895
    }
]
```

this endpoint return an array

- `archived` is a `boolean` value
- `id` is the database id (integer)
- `name` is the repository name (string)
- `star` is the number of stargazers for this repository

###### Filtering repositories

- archived, sould be 0 or 1
- stars, should be an integer -> filter is _greather than or equal_
- name, should be a string -> filter is any name _containing_ the provided value

#### TODO

+ Write tests
+ Add dockerfile
+ Add docker compose
+ Find a way to create a background task (cron)