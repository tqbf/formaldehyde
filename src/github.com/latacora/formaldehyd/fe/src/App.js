import React, { Component } from 'react';
import logo from './logo.svg';
import './App.css';

import BaseComponent from './BaseComponent';
import TextField from './TextField';
import CheckField from './CheckField';
import RadioField from './RadioField';
import SelectField from './SelectField';
import Page from './Page';

import 'grommet-css';

import { server } from './Util';

import App from 'grommet/components/App';
import Split from 'grommet/components/Split';
import Box from 'grommet/components/Box';
import Sidebar from 'grommet/components/Sidebar';
import Header from 'grommet/components/Header';
import Title from 'grommet/components/Title';
import Menu from 'grommet/components/Menu';
import Anchor from 'grommet/components/Anchor';
import User from 'grommet/components/icons/base/User';
import Button from 'grommet/components/Button';
import Footer from 'grommet/components/Footer';
import Section from 'grommet/components/Section';
import FormField from 'grommet/components/FormField';
import TextInput from 'grommet/components/TextInput';

class ReactApp extends BaseComponent {
  state = {
    pages: [],
    tbox: "",
    t2box: "",
    check1: false,
    choices: {
      cfoo: {
        label: "Choice foo",
        checked: false
      },
      cbar: {
        label: "Choice bar",
        checked: false
      },
      cbaz: {
        label: "Choice baz",
        checked: true
      },
      ___current: 'cbaz',
    },
  }

  constructor(props) {
    super(props);
  }

  componentDidMount() {
    server.json.get(`/form/foo`, result => {
      console.log(result);

      this.setState({
        pages: result.children,
      });
    });
  }

  radioChange = (event) => {
    let state = this.state;
    state.choices[state.choices.___current].checked = false;
    state.choices[event.target.id].checked = true;
    state.choices.___current = event.target.id;
    this.setState(state);
  }

  render() {
    console.log(this.state);

    let pages = this.state.pages.map(page => <Page children={ page.children } />);

    return (
      <App>
        <Split flex="right">
          <Sidebar colorIndex='neutral-1'
                   size='small'>
            <Header pad='medium'
                    justify='between'>
              <Title>
                Title
              </Title>
            </Header>
            <Box flex='grow'
                 justify='start'>
              <Menu primary={ true }>
                <Anchor href='#'
                        className='active'>
                  First
                </Anchor>
                <Anchor href='#'>
                  Second
                </Anchor>
                <Anchor href='#'>
                  Third
                </Anchor>
              </Menu>
            </Box>
            <Footer pad='medium'>
              <Button icon={ <User /> } />
            </Footer>
          </Sidebar>
          <Box colorIndex='light-2'
               pad='medium'>
            <Section>
              <TextField value={ this.state.t2box }
                         onChange={ this.change('t2box') }
                         legend="Test field"
                         help="This is field help" />
              <CheckField label="This is a foo"
                          checked={ this.state.check1 }
                          onChange={ this.change('check1') } />
              <SelectField legend="This is a foo"
                           checked={ this.state.check1 }
                           options={ ["one", "dos", "zwei"] }
                           onChange={ this.change('check1') } />
              <RadioField legend="This is a radio"
                          choices={ this.state.choices }
                          onChange={ this.radioChange } />
              <TextField value={ this.state.t2box }
                         onChange={ this.change('t2box') }
                         legend="Test field"
                         height={ 4 }
                         help="This is field melpus" />
            </Section>
            { pages }
          </Box>
        </Split>
      </App>
      );
  }
}

export default ReactApp;
