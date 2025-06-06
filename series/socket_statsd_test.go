package series

// GraphiteBackend returns a SocketBackend pre-configured for Graphite TCP traffic.
// Additional CollectorBackendSocketConf options can be supplied to further
// customize the backend behaviour.
func GraphiteBackend(addr string, opts ...CollectorBakendSocketOptionProvider) (CollectorBackend, error) {
	base := []CollectorBakendSocketOptionProvider{
		CollectorBackendSocketConfWithRenderer(MakeGraphiteRenderer()),
		CollectorBackendSocketConfNetowrkTCP(),
		CollectorBackendSocketConfAddress(addr),
		CollectorBackendSocketConfMessageErrorHandling(CollectorBackendSocketErrorCollect),
		CollectorBackendSocketConfDialErrorHandling(CollectorBackendSocketErrorAbort),
	}
	return SocketBackend(append(base, opts...)...)
}

// StatsdBackend returns a SocketBackend pre-configured for StatsD UDP traffic.
func StatsdBackend(addr string, opts ...CollectorBakendSocketOptionProvider) (CollectorBackend, error) {
	base := []CollectorBakendSocketOptionProvider{
		CollectorBackendSocketConfWithRenderer(MakeStatsdRenderer()),
		CollectorBackendSocketConfNetowrkUDP(),
		CollectorBackendSocketConfAddress(addr),
		CollectorBackendSocketConfMessageErrorHandling(CollectorBackendSocketErrorCollect),
		CollectorBackendSocketConfDialErrorHandling(CollectorBackendSocketErrorAbort),
	}
	return SocketBackend(append(base, opts...)...)
}
