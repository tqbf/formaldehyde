import React, { Component } from 'react';
import PropTypes from 'prop-types';
import logo from './logo.svg';
import './App.css';
import BaseComponent from './BaseComponent';

import 'grommet-css';

import Select from 'grommet/components/Select';
import Box from 'grommet/components/Box';
import Markdown from 'grommet/components/Markdown';
import Paragraph from 'grommet/components/Paragraph';
import Heading from 'grommet/components/Heading';

import TextField from './TextField';

class Textfield extends BaseComponent {
  state = {
    value: this.props.obj.default,
  }

  onChange = (event) => this.setState({
    value: event.target.value
  });


  render() {
    return (
      <div>
        <TextField value={ this.state.value }
                   onChange={ this.onChange }
                   height={ parseInt(this.props.obj.height || "1") }
                   legend={ this.props.obj.label } />
      </div>
      );
  }
}

class RadioField extends BaseComponent {
  state = {
    value: this.props.obj.default,
  }

  onChange = (event) => this.setState({
    value: event.target.value
  });


  render() {
    return (
      <div>
        { JSON.stringify(this.props.obj) }
      </div>
      );
  }
}

const Text = function(props) {
  return (
    <Paragraph size="large">
      { props.obj.text }
    </Paragraph>
    );
}

const Head = function(props) {
  return (
    <Heading strong={ true }>
      { props.obj.text }
    </Heading>
    );
}

export default class Page extends BaseComponent {
  state = {
  }

  constructor(props) {
    super(props);
  }

  render() {
    let es = this.props.children.map((kid) => {
      switch (kid.type) {
        case 'textfield':
          return <Textfield obj={ kid }
                            parent={ this } />;
        case 'text':
          return <Text obj={ kid }
                       parent={ this } />;
        case 'heading':
          return <Head obj={ kid }
                       parent={ this } />;
        case 'radio':
          return <RadioField obj={ kid }
                             parent={ this } />;


        default:
          return (<p>
                    { kid.type }
                  </p>);
      }
    });

    return (
      <div>
        { es }
      </div>
      );
  }
}

Page.propTypes = {
};

Page.defaultProps = {
};
