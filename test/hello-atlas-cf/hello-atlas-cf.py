"""
A first simple Cloud Foundry Flask app

Author: Ian Huston
License: See LICENSE.txt

"""
from flask import Flask
import os
import pprint
import json
from pymongo import MongoClient
from vcap_services import load_from_vcap_services

app = Flask(__name__)

# Get port from environment variable or choose 9099 as local default
port = int(os.getenv("PORT", 9099))

def try_atlas():
    """ For Cloud Foundry, we need to look in the VCAP_SERVICES environment
        this is a json doc like this:
        { <service-name> : [ { name:, credentials:, .. }, .. ], .. }
    """
    result = {}
    try:
        vcap_services = os.getenv('VCAP_SERVICES')
        services = json.loads(vcap_services)
        for service_name in services.keys():
            credentials = load_from_vcap_services(service_name)
            result.update(credentials)
            if 'connectionString' in credentials:
                connection_strings = json.loads(credentials['connectionString'])
                result['connection_strings']=connection_strings
    except Exception as err:
        print( f'Error looking for VCAP_SERVICES {err}')
        result['error']=err
        return result

    mongo_results = {}
    try:
        db = MongoClient( result['uri'] )
        mongo_results["MongoClient"]= f'{db}'
        mongo_results["server_info"]=db.server_info()
    except Exception as err:
        print( f'Error looking for VCAP_SERVICES {err}')
        result['error']=err
    finally:
        result['mongo']=mongo_results

    return result

@app.route('/')
def hello_world():
    print("Hello dare!")
    pprint.PrettyPrinter().pprint( dict(os.environ))
    #'Hello World! I am instance ' + str(os.getenv("CF_INSTANCE_INDEX", 0))
    try_atlas_result = try_atlas()
    return '''
<html>
    <head>
        <title>atlas-osb hello-mongo-env</title>
    </head>
    <body>
        <h1>atlas-osb hello-mongo-env</h1>
        <hr/>
        <h3>try_atlas_result</h3>
        <pre>''' + pprint.pformat(try_atlas_result,indent = 2) + '''</pre>
        <hr/>
        <h3>os.environ</h3>
        <pre>''' + pprint.pformat(dict(os.environ),indent = 2) + '''</pre>
    </body>
</html>'''

if __name__ == '__main__':
    # Run the app, listening on all IPs with our chosen port number
    app.run(host='0.0.0.0', port=port)
