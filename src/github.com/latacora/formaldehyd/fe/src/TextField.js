import React, { Component } from 'react';
import PropTypes from 'prop-types';
import logo from './logo.svg';
import './App.css';
import BaseComponent from './BaseComponent';

import 'grommet-css';

import FormField from 'grommet/components/FormField';
import TextInput from 'grommet/components/TextInput';
import Box from 'grommet/components/Box';

export default class TextField extends BaseComponent {
  constructor(props) {
    super(props);
  }

  render() {
    let inner;
    if (this.props.height > 1) {
      inner = (
        <FormField label={ this.props.help }>
          <textarea rows={ this.props.height }
                    type='text'
                    onChange={ this.props.onChange } />
        </FormField>
      );

    } else {
      inner = (
        <FormField help={ this.props.help }>
          <TextInput id='text-input'
                     value={ this.props.value }
                     onDOMChange={ this.props.onChange } />
        </FormField>
      );
    }

    return (
      <Box pad={ {horizontal: this.props.hpad, vertical: this.props.vpad, between: this.props.bpad} }>
        <legend>
          { this.props.legend }
        </legend>
        { inner }
      </Box>
    )
  }
}

TextField.propTypes = {
  legend: PropTypes.string,
  help: PropTypes.node,
  value: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
  height: PropTypes.number.isRequired,
};

TextField.defaultProps = {
  vpad: "small",
  hpad: "medium",
  bpad: "none",
  height: 1,
};
