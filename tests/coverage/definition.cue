widget: {
	type: "component"
}
template: {
	output: {
		spec: name: parameter.name
	}
	parameter: {
		// +usage=Name.
		name!: string
		size?: int & >=1
		encryption?: {
			enabled!: bool
		}
		replicas?: [...string]
		_validate: {
			"size is required with encryption": true
		}
		_validate: {
			"another rule": true
		}
	}
	#Helper: {
		ignored!: string
	}
}
