// ------------------------------------------------------------ 

// See below, "server", for documentation

// common failure handler for HTTP methods
function srvfail(url) {
  //  snackMessage(`request to server failed. server may be down.`);
}

function fetchJson(url, fn) {
  return fetch(url, {
    'credentials': 'include',
  }).then(function(response) {
    if (response.status >= 400) {
      srvfail(url);
    }
    return response.json();
  }).then(function(data) {
    fn(data);
  }).catch((err) => {
    srvfail(url);
  });
}

export { fetchJson };

// ------------------------------------------------------------ 

function postPut(url, data, fn, method) {
  return fetch(url, {
    method: method,
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data),
    credentials: 'include',
  }).then((r) => {
    if (!r.ok) {
      return r.json().then(err => {
        srvfail(url);
      });
    }
    return r.json();
  }).then((data) => {
    fn(data);
  }).catch((err) => {
    srvfail(url);
  });
}

function postJson(url, data, fn) {
  return postPut(url, data, fn, "POST");
}

export { postJson };

function putJson(url, data, fn) {
  return postPut(url, data, fn, "PUT");
}

export { putJson };

function deleteJson(url, data, fn) {
  return postPut(url, data, fn, "DELETE");
}

export { deleteJson };

// ------------------------------------------------------------ 

function deleteHttp(url, fn) {
  return fetch(url, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json'
    },
    credentials: 'include',
  }).then((r) => {
    if (!r.ok) {
      return r.json().then(err => {
        srvfail(url);
      });
    }
    fn(true);
  }).catch((err) => {
    srvfail(url);
  });
}

export { deleteHttp };

// ------------------------------------------------------------ 

// Preferred server interaction:

// import { server } from './Util';

var server = {
  json: {
    // server.json.get(`/foo/${ props.id }`, result => console.log(result.foo) );

    get: fetchJson,

    // server.json.post(`/foo`, { bar: 1 }, result => console.log(result) );

    post: postJson,

    // server.json.put(`/foo/1`, { bar: 1 }, result => console.log(result) );

    put: putJson,

    // server.json.delete(`/foo/1`, {}, result => console.log(result) );

    delete: deleteJson,
  },

  http: {
    // if you're not going to send JSON data with your DELETE:

    delete: deleteHttp,
  }
};

export { server };

// ------------------------------------------------------------ 

// Quick cheat sheet: 

// componentWillMount()
// render()
// componentDidMount()
// componentWillReceiveProps()
// shouldComponentUpdate()
// componentWillUpdate()
// componentDidUpdate()
// componentWillUnmount()
//  
// setState()
// forceUpdate()
