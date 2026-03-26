glance extension that shows your Todoist tasks.

https://github.com/glanceapp/glance

## config

```yaml
- type: extension
  url: http://localhost:8081
  allow-potentially-dangerous-html: true
  cache: 1m
```

with a filter:

```yaml
- type: extension
  url: "http://localhost:8081?filter=today | overdue"
  allow-potentially-dangerous-html: true
  cache: 1m
```

with a filter and custom collapse:

```yaml
- type: extension
  url: "http://localhost:8081?filter=today | overdue&collapse_after=10"
  allow-potentially-dangerous-html: true
  cache: 1m
```

the `filter` value accepts [todoists filter language](https://todoist.com/help/articles/introduction-to-filters-V98wIH).
