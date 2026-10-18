alter table events drop constraint if exists events_type_check;
alter table events
    add constraint events_type_check
    check (type in ('pageview', 'custom', 'ping', 'link_click', 'pixel'));
