# DataLoader

DataLoader implements a counter which can be used against Field Resolver
functions. It calls a **batch** function after the number of keys (identifiers)
passed to it reaches the loaders capacity.

## Terminology

The following terms are used throughout the documentation:

- **Element** - Refers to the item to be resolved. This can be a record from a
  database table or GraphQL type.

##### Implementation/Usage

Use the DataLoader when fetching a known number of elements, where each element has
a field resolver associated with it that hits a database or has some other time
consuming logic behind resolving the data. This is typically useful when making
_index_ type queries where _n_ number of root elements is requested and each root
element has an associated model to be fetched.

For example, for a `users` table which contains:

- first_name
- last_name
- status_id (foreign key - status table)

and a `status` table which contains:

- id
- value (string)

Performing a query like:

```
{
  users(num: 10) {
    first_name
    status {
      value
    }
  }
}
```

where the `users` resolver returns an array of users (10 in this case). This
will typically result in 1 call to return all 10 users, and 10 calls to resolve
the `status` field for each user.

Use the DataLoader by setting its capacity to _n_ (10 in this case) and
providing a batch loader function which accepts the keys and should return
_n_ number of `status` records. The result of which is a single call to
the database to return the `status` elements.

**Do not use this library when making an unknown number of queries/requests.**

Internally, the DataLoader waits for the `Load(Key)` function to be called _n_ times,
where _n_ is the initial DataLoader capacity. The `Load(Key)` function blocks each
caller until the number of calls equal the loaders capacity and then each call
to `Load(Key)` resolves to the requested element once the batch function returns.

## Future

- caching
  - request level and app level caching.
  - user provided caching scheme (Cacher interface)
- nested resolvers
  - A DataLoader should be provided for a specific field and it should cache the
    results. If a complex query is made (e.g. users have statuses, users have
    todos and todos have the same statuses as users) the loader should:
    - not execute another query if a query is in progress for a specific key,
      use that result instead.
    - batch load the rest of the queries (if the count is known at the time)