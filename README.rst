====================================
``grip`` -- A Golang Logging Library
====================================

Overview
--------

Grip is a high level logging and message system for providing a single
solution for structured logging, notification, and message sending.

#. Provide a common logging interface with support for multiple
   output/messaging backends.

#. Provides tools for collecting structured logging information.

*You just get a grip, folks.*

Use
---

``grip`` declares its dependencies via go modules. The top level ``grip``
package provides global logging functions that use a global logging
interface. You can also use the logging package to produce logger objects with
the same interface to avoid relying on a global logger.

Grip is available under the terms of the Apache License (v2.)

Design
------

Interface
~~~~~~~~~

Grip provides two main interfaces:

- The ``send.Sender`` interfaces which implements sending messages to various
  output sources. Provides sending as well as the ability to support error
  handling, and message formating support.

- The ``message.Composer`` which wraps messages providing both "string"
  formating as well as a "raw data" approach for structured data. With the
  ``message.Base`` implementation, it becomes possible to implement
  ``Composer`` for arbitrary payloads within your application, which may be
  useful for metrics payloads.

Additionally, there are a couple of types for convenience: the ``grip.Logger``
type provides a basic leveled application logging that uses ``Sender``
implementations, and ``message.Builder`` provides a chainable interface for
building log messages.

Goals
~~~~~

- Provide exceptional high-level interfaces for logging and metrics
  collection, with great ergonomics that simplify applications and
  operational stories.

- Integrate with other logging systems (e.g. standard library logging,
  standard output of subprocesses, other libraries, etc.) to accommodate most
  usecases.

- Minimize operational complexity and dependencies for having robust logging
  (e.g. make it possible to log effectively from within a program without
  requiring log relays or collection agents.)

Performance is not explicitly a goal, although reasonable performace should be
possible and architectures should always be possible that prevent the logger
from becoming a performance bottleneck in applications.

Features
--------

Global Logger
~~~~~~~~~~~~~

Following the standard library, and other logging packages, the top-level grip
package has a "global" logger, that you can use without any configuration, and
that wraps the standard library's global logging instance. The global
functions in this package have the same signatures and types as the methods of
the ``Logger`` type. The ``SetGlobalLogger`` allows you to override the global
logger: note, however, that this function (and the functions,) are not
thread-safe when used relative to each other, so *only* use
``SetGlobalLogger`` during process configuration and minimize the amount of
logger re-configuration your application does.

For many applications, using and passing a copy of the ``Logger`` you want to
use is preferable.

``Logger``
~~~~~~~~~~

``Logger`` instances simply wrap ``Sender`` instances and most of the
configuration (e.g. levels and threshold) are actually properties of the
messages and the sender. If you want to create a new logger that is a "child"
of an existing logger, consider something like: ::

    // new logger instance wrapping the sender instance from the global logger
    // this is just for the sake of example:
    logger := grip.NewLogger(grip.Sender())

    // create new logger that annotates all messages
    subLogger := grip.NewLogger(send.MakeAnnotating(
	logger.Sender(),
	map[string]any{
		"module": "http",
		"pid":    os.Getpid(),
	}))


Similarly, you could create an annotating logger that sent to two output
targets: ::

    multiLogger := grip.NewLogger(send.MakeMulti(
	grip.Sender(),
	send.NewAnnotating(
	    logger.Sender(),
	    map[string]any{
		    "module": "http",
		    "pid":    os.Getpid(),
	    })
    ))

While this specific example may not be useful (send all messages to the
standard logger output, and also send message to the annotating sender,) but
this kind of configuration can be useful if you want to filter messages of
different types to different outputs, or write messages to a local output
(e.g. standard output, standard error, system journal, etc.) as well as a
remote service (e.g. splunk, sumo logic, etc.).

The ``x`` Packages
~~~~~~~~~~~~~~~~~~

In an effort to reduce the impact for downstream users of additional
dependencies, the ``x`` package includes code that relies on third party
libraries to provide metrics collecting functionality as well as novel
mechanisms for sending messages. It's possible to use senders and loggers, to
propogate messages over email and sump, as well as using grip as an interface
to send logs directly to external aggregation services, such as splunk.

Features implemented here include:

- sending sumologic/splunk messages directly.
- sending messages directly to syslog and/or the systemd journal.
- desktop notifications
- slack messages
- creating jira tickets and commenting on jira issues
- creating github issues and updating github statuses
- sending email messages
- message payloads the capture system metrics:
  - go runtime metrics
  - process-tree metrics
  - single process metrics.

While the core of grip only has dependency on a single library, `emt
<github.com/tychoish/emt>`_, the packages in the ``x`` hierarchy do have
external dependencies. However, the project and go mod files are structured so
that these libraries are managed by different go mod files and can be
versioned separately.

``send.Sender``
~~~~~~~~~~~~~~~

Senders all wrap some sort of output target, which is at some level an
``io.Writer`` or similar kind of interface. The ``send`` package contains a
number of different configurations (standard error, standard output, files,
etc.) as well as 1additional tools for managing output targets, notably:

- converters for ``Sender`` implementations to ``io.Writer``
  instances.

- connections with standard library logging tools.

- buffering and asynchronous senders to reduce backpressure from loggers and
  to batch workloads to (potentially) slower senders.

- multi sender tools, to manage a group of related outputs.

Senders also permit configurable formating hooks and error handling hooks.

``message.Composer``
~~~~~~~~~~~~~~~~~~~~

The ``Composer`` interface is used for all messages, and provides a flexible
(and simple!) interface to create arbitrary messages, which can be
particularly useful for producing structured logging messages from your
application types. Fundamentally, most ``Composer`` implementations should be lazy,
and require minimal runtime resources in the case that the messages aren't
loggable, either as a result of their content (missing or not rising to the
threshold of loggability,) or because of the priority thresholds on the
logger/sender itself.

The message package provides a collection of implementations and features,
including:

- a ``Base`` type which you can compose in your own ``Composer``
  implementations which provides most of the implementation interface and
  holds some basic message metadata (level, timestamp, pid, hostname.) As a
  result implementors only need to implement ``Loggable``, ``String`` and
  ``Raw`` methods.

- a ``GroupMessage`` that provides a bundle of messages, which sender
  implementations can use to batch output. Additionally, the ``Wrap`` and
  ``Unwrap`` methods provide a stack-based approach to grouping messages.

- the ``Builder`` type provides a chainable interface for creating and sending
  log messages, which is integrated into the ``grip.Logger`` interface.

- Conditional or ``When`` messages allow you to embed logging conditions in
  the message, which can simplify the call site for logging messages.

- Error wrappers that convert go error objects into log messages, which are
  non-loggable when the error is nil, with an error-wrapping function that
  makes it possible to annotate messages.

- Logging functions, or producers, which are functions that produce messages,
  or errors and are only called when the message loggable (e.g. for priority
  level thresholds).

Development
-----------

Future Work
~~~~~~~~~~~

Grip is relatively stable, though there are additional features and areas of
development:

- structured metrics collection. This involves adding a new interface as a
  superset of the Composer interface, and providing ways of filtering these
  messages out to provide better tools for collecting diagnostic data from
  applications.

- additional Sender implementations to support additional output formats and
  needs.

- better integration with recent development in error wrapping in the go
  standard library.

- Shims for other popular logging frameworks to facilitate migrations and
  provide grip users to the benefits of existing infrastructure without
  requiring large refactoring.

If you encounter a problem please feel free to create a github issue or open a
pull request.

History
~~~~~~~

Grip originated as a personal project, and became the default logging and
messaging tool for `Evergreen <https://github.com/evergreen-ci/>`_ and related
projects at MongoDB's release infrastructure developer productivity
organization.

This fork removes some legacy components and drops support older versions of
Golang, thereby adding support for modules. Additionally the ``x`` hierarchy
contains many external integrations that were previously in the main
package. These reorganizations should improve performance and dependency
management and make it easier to stablize releases.
