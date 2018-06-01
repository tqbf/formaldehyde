import React, { Component } from 'react';
import PropTypes from 'prop-types';
import logo from './logo.svg';
import './App.css';
import BaseComponent from './BaseComponent';

import 'grommet-css';

import FormField from 'grommet/components/FormField';
import RadioButton from 'grommet/components/RadioButton';
import Box from 'grommet/components/Box';
import Paragraph from 'grommet/components/Paragraph';

export default class RadioField extends BaseComponent {
  constructor(props) {
    super(props);
  }

  render() {
    const choices = Object.keys(this.props.choices).filter(k => {
      return k.slice(0, 3) != "___"
    }).map(k => {
      const obj = this.props.choices[k];
      return <RadioButton id={ k }
                          label={ obj.label }
                          checked={ obj.checked || false }
                          onChange={ this.props.onChange } />

    });

    return (
      <Box pad={ {horizontal: this.props.hpad, vertical: this.props.vpad, between: this.props.bpad} }>
        <FormField>
          { choices }
        </FormField>
      </Box>
      );
  }
}

RadioField.propTypes = {
  label: PropTypes.string.isRequired,
  toggle: PropTypes.bool,
  reverse: PropTypes.bool,
  onChange: PropTypes.func.isRequired,
  vpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
  hpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
  bpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
};

RadioField.defaultProps = {
  vpad: "small",
  hpad: "medium",
  bpad: "none",
  toggle: false,
  reverse: false,
  checked: false,
};
