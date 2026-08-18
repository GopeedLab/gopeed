(function () {
  const failures = [];
	const pending = [];
  let passed = 0;

  function format(value) {
    if (typeof value === "string") return JSON.stringify(value);
    try { return JSON.stringify(value); } catch (_) { return String(value); }
  }

  function fail(message) {
    throw new Error(message || "assertion failed");
  }

  globalThis.self = globalThis;
  globalThis.setup = function (callback) {
    callback.call({ add_cleanup() {} });
  };
  globalThis.test = function (callback, name) {
	const cleanups = [];
    try {
	  callback.call({ add_cleanup(fn) { cleanups.push(fn); } });
      passed++;
    } catch (error) {
      failures.push((name || "unnamed test") + ": " + (error && error.stack ? error.stack : error));
	} finally {
	  for (let index = cleanups.length - 1; index >= 0; index--) {
		try { cleanups[index](); } catch (_) {}
	  }
    }
  };
  globalThis.promise_test = function (callback, name) {
	const cleanups = [];
	const context = {
	  add_cleanup(fn) { cleanups.push(fn); },
	  step_timeout(fn, delay) { return setTimeout(fn, delay); },
	  unreached_func(message) { return function () { assert_unreached(message); }; },
	};
	const promise = Promise.resolve().then(() => callback.call(context, context)).then(() => {
	  passed++;
	}, (error) => {
	  failures.push((name || "unnamed promise test") + ": " + (error && error.stack ? error.stack : error));
	}).finally(() => {
	  for (let index = cleanups.length - 1; index >= 0; index--) {
		try { cleanups[index](); } catch (_) {}
	  }
	});
	pending.push(promise);
	return promise;
  };
  globalThis.assert_true = function (value, message) {
    if (value !== true) fail((message || "expected true") + "; got " + format(value));
  };
  globalThis.assert_false = function (value, message) {
    if (value !== false) fail((message || "expected false") + "; got " + format(value));
  };
  globalThis.assert_equals = function (actual, expected, message) {
    if (!Object.is(actual, expected)) {
      fail((message || "values differ") + "; expected " + format(expected) + ", got " + format(actual));
    }
  };
  globalThis.assert_not_equals = function (actual, expected, message) {
    if (Object.is(actual, expected)) {
      fail((message || "values should differ") + "; both were " + format(actual));
    }
  };
  globalThis.assert_array_equals = function (actual, expected, message) {
    if (!actual || !expected || actual.length !== expected.length) {
      fail((message || "arrays differ") + "; expected " + format(expected) + ", got " + format(actual));
    }
    for (let index = 0; index < expected.length; index++) {
      if (!Object.is(actual[index], expected[index])) {
        fail((message || "arrays differ") + " at " + index + "; expected " + format(expected) + ", got " + format(actual));
      }
    }
  };
  globalThis.assert_nested_array_equals = function (actual, expected, message) {
    if (!actual || !expected || actual.length !== expected.length) {
      fail((message || "nested arrays differ") + "; expected " + format(expected) + ", got " + format(actual));
    }
    for (let index = 0; index < expected.length; index++) {
      assert_array_equals(actual[index], expected[index], (message || "nested arrays differ") + " at " + index);
    }
  };
  globalThis.assert_unreached = function (message) {
    fail(message || "assert_unreached was reached");
  };
  globalThis.assert_throws_js = function (constructor, callback, message) {
    let thrown = null;
    try { callback(); } catch (error) { thrown = error; }
    if (thrown == null) fail((message || "expected exception") + "; nothing was thrown");
    if (!(thrown instanceof constructor)) {
      fail((message || "wrong exception") + "; expected " + constructor.name + ", got " + thrown);
    }
  };
	globalThis.promise_rejects_js = function (_test, constructor, promise, message) {
	  return Promise.resolve(promise).then(() => {
		fail((message || "expected promise rejection") + "; promise resolved");
	  }, (error) => {
		if (!(error instanceof constructor)) {
		  fail((message || "wrong rejection") + "; expected " + constructor.name + ", got " + error);
		}
	  });
	};
	globalThis.assert_greater_than = function (actual, expected, message) {
	  if (!(actual > expected)) fail((message || "expected greater value") + "; expected > " + expected + ", got " + actual);
	};
  globalThis.__wptFinish = async function () {
	await Promise.all(pending);
    if (failures.length) throw new Error(failures.join("\n\n"));
    return JSON.stringify({ passed, failed: 0 });
  };
})();
