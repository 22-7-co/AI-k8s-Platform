package prometheus

// DefaultFaultQuery is the PromQL used to detect nodes with recent GPU XID errors.
const DefaultFaultQuery = `increase(gpu_xid_errors_total[5m]) > 0`
