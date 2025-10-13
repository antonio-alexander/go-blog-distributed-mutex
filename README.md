# go-blog-distributed-mutex (github.com/antonio-alexander/go-blog-distributed-mutex)

The purpose of this repository is to explore the concept of distributed mutexes from the perspective of data consistency. The _easiest_ mutexes are limited to things like file mutexes and in-memory mutexes and although these work really well, when it comes to situations where you have "more" distribution such as when your mutex must be deployed/used across multiple instances of a given application (e.g. Kubernetes) or where you have to implement it in a database (e.g. multiple clients/applications are interacting with the same table/row) the solution becomes slightly more complicated.

And while in hindsight, the solution isn't necessarily technically complex; there are a number of pros and cons that affect the efficacy, throughput and experience of the solution and these are the things that we'll explore. Upon completion of this repository, you should be able to understand and demonstrate the following:

- how concurrent mutation [of the same data] can affect data consistency
- how concurrent mutation [of the same data] with a mutex has reduced throughput
- how concurrent mutation [of the same data] with versioning has increased error rate
- how concurrent mutation with a distributed (redis) mutex will work across application instances

While I won't say you'll walk away an expert on the topics contained within, you should be able to know if this is the solution to the problem you're trying to solve and enough context to qualify if you've solved the problem in way you consumers will be happy with. This repository will attempt to follow a logical path to ensure that you have enough context to properly implement this solution.

## Bibliography

- [https://dev.to/jdvert/handling-mutexes-in-distributed-systems-with-redis-and-go-5g0d](https://dev.to/jdvert/handling-mutexes-in-distributed-systems-with-redis-and-go-5g0d)
- [https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/)
- [https://medium.com/itnext/solving-double-booking-at-scale-system-design-patterns-from-top-tech-companies-4c5a3311d8ea](https://medium.com/itnext/solving-double-booking-at-scale-system-design-patterns-from-top-tech-companies-4c5a3311d8ea)
- [https://medium.com/@antonio-alexander/data-consistency-the-don-quixote-of-sagas-3603a293d27f](https://medium.com/@antonio-alexander/data-consistency-the-don-quixote-of-sagas-3603a293d27f)
- [https://medium.com/@antonio-alexander/virtual-serialization-when-two-threads-become-one-4e7e4a984acf](https://medium.com/@antonio-alexander/virtual-serialization-when-two-threads-become-one-4e7e4a984acf)
- [https://medium.com/mr-plan-publication/how-to-use-distributed-locks-in-go-to-solve-concurrency-issues-9006d8450a30](https://medium.com/mr-plan-publication/how-to-use-distributed-locks-in-go-to-solve-concurrency-issues-9006d8450a30)

## Getting Started

This repository stands by the idea that a picture is worth 1000 words (seeing is believing); not only is there a complete write-up of how distributed mutexes can be applied, but there are practical examples (written in Go) that show some of the resolutions described in this document. To run the example, you'll need __Docker__ and __Make__. You can execute the following commands and you should see the examples in action:

```sh
make run
```

Once the application runs, you should see output similar to the below:

```log
Configuration:
 mutex: redis
 go routines: 2
 duration: 10s
 interval: 1s

===========================================
--Testing Concurrent Mutate with no Mutex--
===========================================
go routine [0]:
 total mutations: 10
 data inconsistencies: 4
 total errors: 0
go routine [1]:
 total mutations: 10
 data inconsistencies: 6
 total errors: 0

========================================
--Testing Concurrent Mutate with Mutex--
========================================
go routine [0]:
 total mutations: 9
 data inconsistencies: 0
 total errors: 0
go routine [1]:
 total mutations: 9
 data inconsistencies: 0
 total errors: 0

=============================================
--Benchmarking Concurrent Mutate with Mutex--
=============================================
go routine [1]:
 total mutations: 9
 average time: 17.931931ms
 total errors: 0
go routine [0]:
 total mutations: 9
 average time: 19.720474ms
 total errors: 0

===========================================
--Testing Concurrent Mutate with Row Lock--
===========================================
go routine [1]:
 total mutations: 10
 data inconsistencies: 0
 total errors: 0
go routine [0]:
 total mutations: 10
 data inconsistencies: 0
 total errors: 0

================================================
--Benchmarking Concurrent Mutate with Row Lock--
================================================
go routine [0]:
 total mutations: 9
 average time: 16.637586ms
 total errors: 0
go routine [1]:
 total mutations: 10
 average time: 17.280289ms
 total errors: 0

==========================================
--Testing Concurrent Mutate with Version--
==========================================
go routine [0]:
 total mutations: 2
 data inconsistencies: 0
 total errors: 8
go routine [1]:
 total mutations: 8
 data inconsistencies: 0
 total errors: 2

===============================================
--Benchmarking Concurrent Mutate with Version--
===============================================
go routine [0]:
 total mutations: 4
 average time: 14.50463ms
 total errors: 6
go routine [1]:
 total mutations: 6
 average time: 14.104464ms
 total errors: 4
```

## TLDR; Too Long, Didn't Read

> Disclaimer: this is a lot and its super condensed

Distributed mutexes can be used to solve issues involving data consistency when you must scale across application (or file system) boundaries. Mutexes maintain data consistency by ensuring that only one entity can execute a process at any time (assuming that all entities use the mutex). Due to the virtual serialization that occurs when using any kind of mutex, whether an actual scalar mutex or something more complicated like a queue, your throughput will be reduced in a non-trivial way.

These are solutions that employ pessimistic locking (via a distributed mutex):

- redis can be used to create a distributed mutex through use of the set if not exists functionality (if it already exists, the mutex is locked) and a LUA script to delete if exists atomically
- a database can be used to create a distributed mutex through use of row locking (a simple mutex) and take advantage of the high-availability options of the database (e.g. sharding etc.)

These are solutions that employ optimistic locking:

- a database can be used to create a distributed mutex through use of row locking (a read/write mutex)
- a database query that uses an atomically incrementing version read prior to mutation to detect if the row was mutated between the initial read and the final mutation

With the above solutions, we can determine the following:

- concurrent mutation of the same row without a mutex can cause data inconsistency
- concurrent mutation of the same row with a mutex has reduced throughput
- concurrent mutation of the same row with versioning has increased error rate
- concurrent mutation with a distributed mutex (e.g., redis, zookeeper, consul, etc.) will work across application instances

> There's no functional difference between two instances of a given application on the same bare metal or different instances running on two different ec2 instances; _across_ application boundaries are the same. Alternatively a file lock could be used if those instances have access to the same filesystem

## Identifying the Problem

The purpose of this section is to answer the question, "Will this solution solve my problem?" and to do that, we have to contextualize the situation. The answers to the questions below will give some shape to your problem:

- Does your application interact with data that has heavy concurrent mutation on the same/related data?

> Some common examples are things like theatre seats (i.e. the double booking problem) or similarly, but less risky, stock of a given product. When the _thing_ is available for purchase, many people are attempting (concurrently) to buy (mutate) the same thing and because there's only one of those things, if you sell it to two people; you have a problem

- Does your application scale across application (memory) or physical (filesystem) boundaries?

> This could also be re-phrased as, "Is the data or process in question localized"? Using the previous example, if you could purchase those theatre tickets through a kiosk; if there was only one kiosk, the efficacy of your problem is severely limited, but say if your kiosk were a website instead of a physical thing, your solution for mutual exclusion would be significantly different (i.e. a distributed mutex vs an in-memory mutex)

- Does data inconsistency break your application's business logic?

> This tries to contextualize how data inconsistency matters to your business and whether you should take the time/effort to fix it. Data inconsistency could appear as someone's changes being squashed, for example: two people could change the description of an item and they could probably notice almost immediately that their change didn't take and they could fix it. How likely is that to occur? Is that something you could prevent through access controls or ui/ux (maybe they chose the wrong item)? If it's unlikely to happen and the outcome is benign, why go through the trouble?

- How tolerant are your consumers for application throughput?

> How likely are your consumers to notice that something is slow, is the expectation that mutations occur quickly and correctly (e.g., you're a financial institution); or is correct and slow ok (e.g., a health provider) or that they can be incorrect and slow (e.g., a public works). If your consumers won't use the product if its slow, you'll need to choose a solution that's very fast, but may have a high failure rate

- How difficult is it for your application to detect and resolve data inconsistencies?

> This is pretty obvious with financial institutions, how difficult would it be for them to know if money went in the wrong account and how easy would it be for them to resolve it? Sometimes, data inconsistencies will eventually resolve themselves or are impractical (e.g. all operations are idempotent) and in those cases, the efficacy of a mutual exclusion solution is also impractical

- How resilient is your business logic against data inconsistencies?

> This could be re-phrased as, "If enough data inconsistencies occurred, could you stay in business?" If someone figured out a way to generate a license key for your application, would that put you out of business? or is the thing you're selling the service/convenience such that having the application and a license key wouldn't give your consumers enough value. Maybe your business logic and model are so resilient that data being inconsistent doesn't "cost" as much as the resolutions (e.g. customers care that it's very fast and mostly correct and are ok with it not being correct sometimes as long as its fast)

The answers to the above questions can help qualify your risk; in general, if your risk is low you can omit the implementation altogether or choose one that's the most convenient to implement, but if your risk is high, you can choose the solution that best fits your use case; often there may only be one.

![there can only be one](./docs/_images/highlander.gif)

> In an odd way, software development gives the illusion that there are infinite solutions to problems, but the reality is that there are far fewer practical solutions than there are solutions when you have a reasonable amount of context

## Contextualizing Abused Terms/Phrases

I want to define a few terms/phrases that I've abused (and will continue to do so); so that they're not misunderstood and the context of which I'm using them is well understood. Feel free to skip this section if you get the terms and the context:

- data consistency: although I do a more thorough job of explaining data consistency in this repo: [https://github.com/antonio-alexander/go-blog-data-consistency](https://github.com/antonio-alexander/go-blog-data-consistency); the general idea is that if you mutate data fast enough you could cause a race condition where the data is no longer consistent from a practical perspective whether that's data loss or a relationship that doesn't make any sense
- concurrent mutation: this has to do with attempting to write data semi-simultaneously; although I don't state it explicitly I'm _almost_ ALWAYS referring to modifying the same/related data and not just writing data in general
- idempotency: this is the idea that some operations, no matter the order you do them, will always end up with the same data; although idempotency doesn't guarantee data consistency, sometimes it can mean that you maintain it from a practical perspective (i.e., the way we're confirming data-consistency may fail, but the data wouldn't be _wrong_)
- mutual exclusion: (to be abbreviated as mutex) is a tool we can wrap around a process to ensure that only one entity can work on it at a time. See: [https://en.wikipedia.org/wiki/Lock_(computer_science)](https://en.wikipedia.org/wiki/Lock_(computer_science))
- concurrency vs parallelism: this is academic (especially from the perspective of Go); concurrency means happening at about the same time (but maybe not really) while parallelism means happening at the exact same time (often requiring a number of cores greater or equal to those parallel processes). Practically, it doesn't matter, but I'll adhere to concurrent mutation and not parallel mutation because a parallel operation is concurrent but a concurrent operation isn't parallel
- throughput: this is describing how fast an operation can occur under practical circumstances; it's not to be confused with bandwidth which focuses on the theoretical maximum of a given resource

## Throughput and Concurrent Mutation

Throughput is heavily influenced by how long you have to wait on a resource: how long it takes for the network call to complete, how long it takes for the disk io to finish, how long it takes to update the database etc. Although not all time is spent waiting, the less waiting you do the faster the operation will be. If you can minimize waiting and optimize said resources, you'll probably experience a higher throughput than before.

When introducing a mutex into the equation, you affect throughput because you're introducing a process that forces waiting. If a resource is locked behind the mutex, it can't be accessed until it's available therefore your maximum __throughput__ is reduced. So, lets say that mutating data takes 100ms, if two operations attempting to mutate data concurrently, the actual time to mutate the data could be ~100ms. BUT, if there's a mutex preventing concurrent mutation, then it would take at least 200ms. A mutex can "virtually serialize" processes that otherwise would occur independently.

> I wrote a pretty nice blog about the idea of virtual serialization: [https://antonio-alexander.medium.com/virtual-serialization-when-two-threads-become-one-4e7e4a984acf](https://antonio-alexander.medium.com/virtual-serialization-when-two-threads-become-one-4e7e4a984acf). It focuses more on race conditions and synchronization than mutexes, but the ideas support this repository

## Regarding the Implementations

[I think] the implementations within this repository are relatively opinionated, but should provide enough context to be able to implement your own solution (or copy+paste my own). The application that's executed during _make run_ is located in [./internal/main.go](./internal/main.go); this application will attempt to quantify data inconsistency by locking a mutex, reading an employee and then mutating that employee and confirming the version increments only be one:

```go
func employeeCurrentMutateWithMutexDemo(config *Configuration, db *sql.DB, mu Mutex, 
    chOsSignal chan (os.Signal), employee *Employee) error {
    fmt.Println("\n========================================")
    fmt.Println("--Testing Concurrent Mutate with Mutex--")
    fmt.Println("========================================")
    return employeeConcurrentMutateDemo(config, chOsSignal, func(goRoutine, dataInconsistencies int) (int, error) {
        mu.Lock()
        defer mu.Unlock()

        employeeRead, err := ReadEmployee(db, employee.EmailAddress)
        if err != nil {
            return dataInconsistencies, err
        }
        employeeUpdated, err := UpdateEmployee(db, employee)
        if err != nil {
            return dataInconsistencies, err
        }
        if employeeUpdated.Version != employeeRead.Version+1 {
            dataInconsistencies++
        }
        return dataInconsistencies, nil
    })
}
```

It's easy to determine if a data inconsistency (e.g., a race condition) has occurred if the value of version __changes__ such that the version returned after update is NOT the version read + 1.

> In Go, it's convention to lock and then defer unlock, this is a pattern that ensures you don't forget to unlock and upon panic (where execution goes up the call stack) or return, the mutex is always unlocked; this convention makes it unlikely that you'll forget to unlock a mutex, lock a locked mutex or unlock an unlocked mutex; for more about defer, see: [https://gobyexample.com/defer](https://gobyexample.com/defer)

We capture the following to communicate the resolutions described in the [TLDR](#tldr-too-long-didnt-read):

- total mutations: this describes the total number of mutations that were attempted within a specific demo
- data inconsistencies: this describes the total number of data inconsistencies identified
- total errors: this is the total number of errors that have occurred (error rate)
- average time: this is the average time to complete a mutation (throughput)

Looking at the logs, you should notice the following:

- concurrent mutate with no mutex:
  - zero error rate (red herring/implementation issue)
  - non-zero data inconsistencies
- concurrent mutate with a mutex:
  - no data inconsistencies
  - zero error rate
  - throughput is ~18ms
- concurrent mutate with row lock:
  - no data inconsistencies
  - zero error rate
  - throughput is ~16ms
- concurrent mutation with versioning:
  - fewer successful mutations
  - increased error rate
  - no data inconsistencies
  - throughput is ~14ms (faster in comparison to using mutex)

> If it's not obvious, there's no reason to use a mutex AND a row lock

I think it goes without saying that your mileage (especially with throughput) will vary; more resources equals lower throughput additionally this __ONLY__ affects concurrent mutation on the __SAME__ object; more dispersed concurrent mutations (on different objects) will have reduced contention.

## Frequently Asked Questions

Here are a few thought provoking questions that you may ask:

- Why did you use a LUA script to unlock the redis mutex?

> It's not obvious, but to "unlock" a mutex, keeping with the SETNX function, the mutex is only unlocked if the mutex exists and you successfully delete it. If you don't do this within a LUA script it's not considered atomic and thus not safe for concurrent usage.

- What if I want to implement something akin to a row lock, but with redis?

> Assuming it's obvious that you have to change the function prototype for Lock()/Unlock(), you'll need to provide a string (I suggest using a natural key of the row) that's easy to re-create with the data you're locking (e.g., if I was locking a row attached to an employee, their email address would be the natural key, so I’d use that as the mutex key)

- If I create a mutex for each object in Redis, how would that affect throughput?

> It would be identical to using MySQL, rows with high concurrent mutations would have reduced throughput due to the locking and serialization of those mutations

- Is Redis faster than a database?

> Yes, but sometimes you still have to interact with the database and it's less complex and sometimes faster, to simply use the database. The database is often the source of truth while redis (or some distributed cache) is often not, so if it's not data you can reasonably cache, then even if redis is faster, you may not be able to use it. See: [https://github.com/antonio-alexander/go-blog-cache](https://github.com/antonio-alexander/go-blog-cache)

- How to handle situations where distributed mutexes are lost or aren't unlocked?

> The general pattern for redis mutexes are to define them with a TTL (time to live) such that if enough time passes, it's automatically unlocked by timing out (and deleting the key); this is a double-edged sword in that you have to know if you need to _extend_ your TTL for the mutex so it isn't automatically unlocked

- What are situations where we use redis over a database (or vice versa)?

> Not all applications have processes that need a mutex to wrap a process and don't occur wholly inside a database, as a result, if you don't have to interact with a database, redis is a much better tool than a database; if you just need a mutex, redis takes much less effort to implement than a database

- How to handle deadlocks in sql?

> While this shouldn't be a common problem, it does (and can) happen, generally the idea is that you should rollback the transaction and retry after a random wait. Deadlocks will generally return an error as opposed to having to "wait" for it to timeout. Read this link for more information: [https://en.wikipedia.org/wiki/Record_locking](https://en.wikipedia.org/wiki/Record_locking)

- What about high-availability with sql?

> Row locks follow the same rules as replication, as far as I know, you don't have to do anything special with your queries to use them if you have a single instance of sql or multiple instances of sql (reduced complexity)

- What is Redsync?

> a distributed locking algorithm (by redis) that's a bit more complicated than simple locking, it has a ready-made go module that's pretty easy to implement, see: [https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/); this is meant to handle situations where you could hypothetically lock the same thing twice on two connected instances of redis

- I share a Redis instance with other applications, how can I protect my distributed mutex?

> Redis offers ACLs that can be used to restrict access to specific keys or more granularly specific commands on specific keys, you may have to disable the default user or at least restrict its access, but generally its doable. Like anything security related, performing some negative tests will ensure it's working as intended. The setup used for this repository is present in [./config/redis.conf](./config/redis.conf)

- Is there a situation where you would intentionally combine the solutions described in this repository?

> Yes, while at face value, it doesn't make sense to use __two__ mutexes, if they're protecting different things, especially in a nested manner, then it makes total sense; say for example, you implement a row lock to maintain data consistency for an employee, but you have business logic that modifies multiple employees at once and you need to protect _that_ process, you'd have a mutex for the process, and then additinoal mutexes (in the form of row locks) for the items being modified; additionally, there's still an opportunity for concurrent mutation outside of the business logic mutex, so you'd want them in both
