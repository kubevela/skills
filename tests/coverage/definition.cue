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
	}
	#Helper: {
		ignored!: string
	}
}
