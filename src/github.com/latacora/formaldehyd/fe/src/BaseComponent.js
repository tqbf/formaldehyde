import React, { Component } from 'react';
import { fetchJson, postJson } from './Util';
//import { navigate } from './Router';

export default class BaseComponent extends Component {
  constructor(props) {
    super(props);
    this.tokens = [];
  }

  change(key) {
    return (e) => {
      let val;
      if (typeof (this.state[key]) == 'boolean') {
        val = e.target.checked;
      } else {
        val = e.target.value;
      }

      let o = {};
      o[key] = val;
      this.setState(o);
    };
  }

  emit(kind, obj) {
    window.eventSource.emit(kind, obj);
  }

  subscribe(kind, fn) {
    let token = window.eventSource.addListener(kind, fn);
    this.tokens.push(token);
  }

  componentWillUnmount() {
    this.tokens.forEach((t) => {
      t.remove();
    });
  }

  //navigate(url) {
  //  navigate(url);
  //}

}

