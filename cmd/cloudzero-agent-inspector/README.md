# CloudZero Agent Inspector

The CloudZero Agent Inspector is a tool that helps you diagnose errors and misconfigurations in your CloudZero Agent configuration.

## Usage

The easiest way to use the CloudZero Agent Inspector is to use the [CloudZero Agent Helm chart](https://github.com/Cloudzero/cloudzero-charts/tree/develop/charts/cloudzero-agent).

However, you can also run the CloudZero Agent Inspector directly from the binary. By default, it will listen on port 9376 and forward all requests to `https://api.cloudzero.com`, though this can be overridden by command line arguments.

To run the CloudZero Agent Inspector, simply run the executable. Any requests made to the inspector will then be forwarded to the CloudZero API. If the inspector detects errors it will log a description of the error to the console. For common errors, such as an invalid API key, the inspector will include a human-friendly description of the problem.
